import time
import uuid
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone

from mhpos_seed import (
    call,
    create_cloud_seed,
    create_post_pairing_sync_item,
    get_edge_node_device_id,
    health_check,
    provision_pos_edge,
    redacted_summary,
    stamp_summary,
    verify_pos_read_model,
    verify_sync_status,
    wait_for_menu_item,
)


STATUS_PASSED = "passed"
STATUS_FAILED = "failed"
STATUS_SKIPPED = "skipped"
ALL_SUITES = ["health", "license_pairing", "cloud_to_edge_masterdata"]


@dataclass
class StackContext:
    cloud: object
    pos: object
    license: object
    cloud_base_url: str = "http://localhost:8090"
    pos_base_url: str = "http://localhost:8080"
    license_base_url: str = "http://localhost:8095"


def parse_suites(values):
    if not values:
        return list(ALL_SUITES)
    out = []
    for raw in values:
        for item in str(raw).split(","):
            name = item.strip()
            if not name:
                continue
            if name == "all":
                for suite in ALL_SUITES:
                    if suite not in out:
                        out.append(suite)
                continue
            if name not in ALL_SUITES:
                raise ValueError(f"unknown stack smoke suite {name}")
            if name not in out:
                out.append(name)
    return out or list(ALL_SUITES)


def passed_result(name, details=None, started=None):
    return check_result(name, STATUS_PASSED, details or {}, started=started)


def failed_result(name, error, details=None, started=None):
    return check_result(name, STATUS_FAILED, details or {}, error=str(error), started=started)


def skipped_result(name, reason, details=None, started=None):
    return check_result(name, STATUS_SKIPPED, details or {}, error=str(reason), started=started)


def check_result(name, status, details, error="", started=None):
    started = started or time.monotonic()
    return {
        "name": name,
        "status": status,
        "duration_ms": int((time.monotonic() - started) * 1000),
        "details": safe_value(details),
        "error": str(error or ""),
    }


def stack_result(suites, artifacts=None):
    status = STATUS_PASSED
    if any(item["status"] == STATUS_FAILED for item in suites):
        status = STATUS_FAILED
    elif any(item["status"] == STATUS_SKIPPED for item in suites):
        status = STATUS_SKIPPED
    result = {
        "status": status,
        "generated_at_unix": int(time.time()),
        "suites": suites,
    }
    if artifacts:
        result["_artifacts"] = artifacts
    return result


def safe_value(value):
    if isinstance(value, dict):
        out = {}
        for key, item in value.items():
            if key in ("pin", "cashier_pin", "manager_pin", "pairing_code", "token", "credentials"):
                out[key] = "<redacted>"
            else:
                out[key] = safe_value(item)
        return out
    if isinstance(value, list):
        return [safe_value(item) for item in value]
    return value


def run_health_suite(ctx):
    started = time.monotonic()
    details = {}
    failed = []
    for name, client in (("cloud", ctx.cloud), ("pos", ctx.pos), ("license", ctx.license)):
        service_started = time.monotonic()
        try:
            details[name] = {
                "status": STATUS_PASSED,
                "duration_ms": int((time.monotonic() - service_started) * 1000),
                "response": health_check(client),
            }
        except Exception as exc:
            failed.append(name)
            details[name] = {
                "status": STATUS_FAILED,
                "duration_ms": int((time.monotonic() - service_started) * 1000),
                "error": str(exc),
            }
    if failed:
        return failed_result("health", "health failed for " + ", ".join(failed), details, started=started)
    return passed_result("health", details, started=started)


def run_license_pairing_suite(ctx):
    started = time.monotonic()
    suffix = uuid.uuid4().hex[:10]
    pairing_code = "PY" + uuid.uuid4().hex[:10].upper()
    node_device_id = "license-smoke-node-" + suffix
    restaurant_id = "license-smoke-restaurant-" + suffix
    expires_at = (datetime.now(timezone.utc) + timedelta(minutes=15)).isoformat().replace("+00:00", "Z")
    body = {
        "pairing_code": pairing_code,
        "cloud_url": ctx.cloud_base_url,
        "restaurant_id": restaurant_id,
        "node_device_id": node_device_id,
        "credentials": {"type": "node_token", "token": "stack-smoke-token-" + suffix},
        "expires_at": expires_at,
    }
    registered = call(ctx.license, "registerLicensePairingCode", body)
    resolved = call(
        ctx.license,
        "resolveLicensePairingCode",
        {"pairing_code": pairing_code, "node_device_id": node_device_id},
    )
    if resolved.get("restaurant_id") != restaurant_id or resolved.get("node_device_id") != node_device_id:
        raise AssertionError("license resolve returned unexpected restaurant_id or node_device_id")
    second_rejected = False
    try:
        call(
            ctx.license,
            "resolveLicensePairingCode",
            {"pairing_code": pairing_code, "node_device_id": node_device_id},
        )
    except Exception:
        second_rejected = True
    if not second_rejected:
        raise AssertionError("license pairing code was not consumed after first resolve")
    return passed_result(
        "license_pairing",
        {
            "registered_status": registered.get("status", "registered"),
            "restaurant_id": restaurant_id,
            "node_device_id": node_device_id,
            "second_resolve_rejected": second_rejected,
        },
        started=started,
    )


def run_cloud_to_edge_masterdata_suite(
    ctx,
    restaurant_name="",
    cashier_pin="1111",
    manager_pin="2222",
    node_device_id="",
    suffix="",
    client_device_id="python-stack-smoke-client",
    skip_post_pairing_sync_check=False,
    wait_seconds=90,
    interval_seconds=2,
    existing_summary=None,
):
    started = time.monotonic()
    provisioning_status = call(ctx.pos, "getProvisioningStatus")
    if provisioning_status.get("paired"):
        summary = matching_existing_summary(existing_summary, provisioning_status)
        return verify_cloud_to_edge_masterdata(
            ctx,
            summary,
            client_device_id=client_device_id,
            skip_post_pairing_sync_check=skip_post_pairing_sync_check,
            wait_seconds=wait_seconds,
            interval_seconds=interval_seconds,
            started=started,
            reused_existing_pairing=True,
        )
    node_device_id = node_device_id or provisioning_status.get("node_device_id") or get_edge_node_device_id(ctx.pos)
    summary = create_cloud_seed(
        ctx.cloud,
        restaurant_name=restaurant_name,
        cashier_pin=cashier_pin,
        manager_pin=manager_pin,
        node_device_id=node_device_id,
        suffix=suffix,
    )
    provisioning = provision_pos_edge(ctx.cloud, ctx.pos, ctx.cloud_base_url, summary["restaurant_id"], node_device_id)
    summary.update(provisioning)
    summary = stamp_summary(summary, ctx.cloud_base_url, ctx.pos_base_url)
    return verify_cloud_to_edge_masterdata(
        ctx,
        summary,
        client_device_id=client_device_id,
        skip_post_pairing_sync_check=skip_post_pairing_sync_check,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
        started=started,
        reused_existing_pairing=False,
    )


def matching_existing_summary(existing_summary, provisioning_status):
    if not existing_summary:
        raise RuntimeError(
            "POS Edge is already paired with restaurant "
            + str(provisioning_status.get("restaurant_id", ""))
            + "; provide a matching seed summary or reset local stack data before running cloud_to_edge_masterdata"
        )
    restaurant_id = provisioning_status.get("restaurant_id", "")
    node_device_id = provisioning_status.get("node_device_id", "")
    if existing_summary.get("restaurant_id") != restaurant_id:
        raise RuntimeError(
            "POS Edge is already paired with restaurant "
            + str(restaurant_id)
            + ", but seed summary belongs to "
            + str(existing_summary.get("restaurant_id", ""))
        )
    if node_device_id and existing_summary.get("node_device_id") and existing_summary.get("node_device_id") != node_device_id:
        raise RuntimeError(
            "POS Edge node_device_id "
            + str(node_device_id)
            + " does not match seed summary node_device_id "
            + str(existing_summary.get("node_device_id", ""))
        )
    if not existing_summary.get("manager_pin"):
        raise RuntimeError("seed summary does not contain manager_pin required for POS verification")
    return dict(existing_summary)


def verify_cloud_to_edge_masterdata(
    ctx,
    summary,
    client_device_id,
    skip_post_pairing_sync_check,
    wait_seconds,
    interval_seconds,
    started,
    reused_existing_pairing,
):
    initial_read = verify_pos_read_model(ctx.pos, summary, client_device_id=client_device_id)
    post_pairing = None
    synced_item = None
    if not skip_post_pairing_sync_check:
        post_pairing = create_post_pairing_sync_item(ctx.cloud, summary)
        try:
            synced_item = wait_for_menu_item(
                ctx.pos,
                post_pairing["menu_item_id"],
                initial_read["headers"],
                timeout_seconds=wait_seconds,
                interval_seconds=interval_seconds,
            )
        except TimeoutError as exc:
            sync_status = verify_sync_status(ctx.pos, initial_read["headers"])
            raise RuntimeError(post_pairing_timeout_message(sync_status)) from exc
        summary["post_pairing_sync_menu_item_id"] = post_pairing["menu_item_id"]
    sync_status = verify_sync_status(ctx.pos, initial_read["headers"])
    result = passed_result(
        "cloud_to_edge_masterdata",
        {
            "reused_existing_pairing": reused_existing_pairing,
            "summary": redacted_summary(summary),
            "initial_read_model": {k: v for k, v in initial_read.items() if k != "headers"},
            "post_pairing_sync": post_pairing,
            "synced_item": synced_item,
            "sync": sync_status,
        },
        started=started,
    )
    result["_artifacts"] = {"masterdata_summary": dict(summary)}
    return result


def post_pairing_timeout_message(sync_status):
    status = sync_status.get("sync_status", {}) if isinstance(sync_status, dict) else {}
    outbox = sync_status.get("outbox", []) if isinstance(sync_status, dict) else []
    last_error = ""
    if isinstance(outbox, list):
        fallback_error = ""
        for item in outbox:
            if isinstance(item, dict) and item.get("last_error"):
                candidate = str(item.get("last_error"))
                if not fallback_error:
                    fallback_error = candidate
                if "sync direction blocked" not in candidate and "local_only" not in candidate:
                    last_error = candidate
                    break
        if not last_error:
            last_error = fallback_error
    parts = [
        "post-pairing Cloud->Edge menu item did not sync before timeout",
        "pending=" + str(status.get("pending", "")),
        "failed=" + str(status.get("failed", "")),
        "suspended=" + str(status.get("suspended", "")),
    ]
    if last_error:
        parts.append("last_error=" + last_error)
    return "; ".join(parts)


def run_selected_suites(ctx, suites, **options):
    results = []
    artifacts = {}
    for suite in suites:
        started = time.monotonic()
        try:
            if suite == "health":
                result = run_health_suite(ctx)
            elif suite == "license_pairing":
                result = run_license_pairing_suite(ctx)
            elif suite == "cloud_to_edge_masterdata":
                result = run_cloud_to_edge_masterdata_suite(ctx, **options)
            else:
                result = skipped_result(suite, "unknown suite", started=started)
        except Exception as exc:
            result = failed_result(suite, exc, started=started)
        suite_artifacts = result.pop("_artifacts", {})
        artifacts.update(suite_artifacts)
        results.append(result)
    return stack_result(results, artifacts=artifacts)
