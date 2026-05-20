import time
import uuid
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone

from mhpos_seed import (
    auth_headers,
    call,
    create_cloud_seed,
    create_post_pairing_sync_item,
    get_edge_node_device_id,
    health_check,
    login_with_pin,
    provision_pos_edge,
    redacted_summary,
    stamp_summary,
    verify_pos_read_model,
    verify_sync_status,
    wait_for_menu_item,
)
from mhpos_http import HttpError


STATUS_PASSED = "passed"
STATUS_FAILED = "failed"
STATUS_SKIPPED = "skipped"
ALL_SUITES = [
    "health",
    "license_pairing",
    "cloud_to_edge_masterdata",
    "pos_cashier_runtime",
    "pos_refund_after_shift_close",
]


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
            safe_key = str(key).lower()
            if safe_key in (
                "pin",
                "cashier_pin",
                "manager_pin",
                "manager_pin_hash",
                "pairing_code",
                "token",
                "credentials",
                "session",
                "session_id",
                "x-session-id",
                "authorization",
                "cookie",
                "headers",
            ):
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


def run_pos_cashier_runtime_suite(
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
    summary = ensure_pos_runtime_summary(
        ctx,
        restaurant_name=restaurant_name,
        cashier_pin=cashier_pin,
        manager_pin=manager_pin,
        node_device_id=node_device_id,
        suffix=suffix,
        client_device_id=client_device_id,
        skip_post_pairing_sync_check=skip_post_pairing_sync_check,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
        existing_summary=existing_summary,
    )
    session = prepare_pos_runtime_session(ctx, summary, client_device_id)
    sale = create_pos_runtime_sale(ctx, summary, session)
    employee_shift = session["employee_shift"]
    runtime_headers = session["runtime_headers"]
    manager_headers = session["manager_headers"]
    manager_employee_id = session["manager_employee_id"]
    order_id = sale["order_id"]
    check_id = sale["check_id"]
    closed_orders = call(
        ctx.pos,
        "listClosedOrders",
        query={"shift_id": employee_shift["id"], "device_id": summary["node_device_id"], "limit": 10},
        headers=runtime_headers,
    )
    if not find_by_id(closed_orders, order_id):
        raise AssertionError("bounded closed orders read did not include the paid order")
    check = call(ctx.pos, "getCheck", path_params={"id": check_id}, headers=runtime_headers)
    reprint = call(
        ctx.pos,
        "reprintCheck",
        {"command_id": new_command_id("reprint-check")},
        path_params={"id": check_id},
        headers=manager_headers,
    )
    cancellation = call(
        ctx.pos,
        "recordCheckCancellation",
        {
            "command_id": new_command_id("record-cancellation"),
            "operation_kind": "full",
            "inventory_disposition": "no_stock_effect",
            "reason": "stack smoke same-shift cancellation",
            "approved_by_employee_id": manager_employee_id,
        },
        path_params={"id": check_id},
        headers=manager_headers,
    )
    operations = call(ctx.pos, "listCheckFinancialOperations", path_params={"id": check_id}, headers=manager_headers)
    if not any(item.get("id") == cancellation.get("id") and item.get("operation_type") == "cancellation" for item in operations):
        raise AssertionError("check financial operations did not include the recorded cancellation")
    storage_status = call(ctx.pos, "getStorageStatus", headers=manager_headers)
    result = passed_result(
        "pos_cashier_runtime",
        {
            "reused_existing_pairing": bool(existing_summary),
            "runtime_actor_role": session["runtime_role"],
            "runtime_actor_employee_id": session["runtime_employee_id"],
            "manager_employee_id": manager_employee_id,
            "shift_id": employee_shift.get("id"),
            "cash_shift_id": session["cash_shift"].get("id"),
            "order_id": order_id,
            "line_count": sale["line_count"],
            "normal_line_id": sale["normal_line_id"],
            "modifier_line_added": sale["modifier_line_added"],
            "service_line_added": sale["service_line_added"],
            "precheck_id": sale["precheck_id"],
            "payment_id": sale["payment_id"],
            "payment_status": sale["payment_status"],
            "check_id": check_id,
            "check_status": check.get("status"),
            "reprint_document_type": reprint.get("document_type"),
            "cancellation_operation_id": cancellation.get("id"),
            "financial_operations_count": len(operations),
            "closed_orders_count": len(closed_orders),
            "storage": storage_status_details(storage_status),
        },
        started=started,
    )
    result["_artifacts"] = {"masterdata_summary": dict(summary)}
    return result


def run_pos_refund_after_shift_close_suite(
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
    summary = ensure_pos_runtime_summary(
        ctx,
        restaurant_name=restaurant_name,
        cashier_pin=cashier_pin,
        manager_pin=manager_pin,
        node_device_id=node_device_id,
        suffix=suffix,
        client_device_id=client_device_id,
        skip_post_pairing_sync_check=skip_post_pairing_sync_check,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
        existing_summary=existing_summary,
    )
    session = prepare_pos_runtime_session(ctx, summary, client_device_id)
    sale = create_pos_runtime_sale(ctx, summary, session)
    original_employee_shift = session["employee_shift"]
    original_cash_shift = session["cash_shift"]
    manager_headers = session["manager_headers"]
    manager_employee_id = session["manager_employee_id"]
    runtime_headers = session["runtime_headers"]
    runtime_employee_id = session["runtime_employee_id"]

    closed_cash_shift = call(
        ctx.pos,
        "closeCashShift",
        {
            "command_id": new_command_id("close-original-cash-shift"),
            "closed_by_employee_id": manager_employee_id,
            "closing_cash_amount": 0,
        },
        path_params={"id": original_cash_shift["id"]},
        headers=manager_headers,
    )
    closed_employee_shift = call(
        ctx.pos,
        "closeEmployeeShift",
        {
            "command_id": new_command_id("close-original-employee-shift"),
            "closed_by_employee_id": runtime_employee_id,
        },
        path_params={"id": original_employee_shift["id"]},
        headers=runtime_headers,
    )

    refund_employee_shift = ensure_employee_shift(ctx.pos, summary, manager_employee_id, manager_headers)
    current_cash_shift = optional_call(ctx.pos, "getCurrentCashShift", headers=manager_headers)
    if current_cash_shift:
        if current_cash_shift.get("shift_id") != refund_employee_shift.get("id"):
            raise RuntimeError("open cash shift does not match the refund employee shift for smoke actor")
        refund_cash_shift = current_cash_shift
    else:
        refund_cash_shift = call(
            ctx.pos,
            "openCashShift",
            {
                "command_id": new_command_id("open-refund-cash-shift"),
                "restaurant_id": summary["restaurant_id"],
                "opened_by_employee_id": manager_employee_id,
                "opening_cash_amount": 0,
            },
            headers=manager_headers,
        )

    refund = call(
        ctx.pos,
        "recordCheckRefund",
        {
            "command_id": new_command_id("record-refund-after-shift-close"),
            "operation_kind": "full",
            "inventory_disposition": "no_stock_effect",
            "reason": "stack smoke refund after shift close",
            "approved_by_employee_id": manager_employee_id,
        },
        path_params={"id": sale["check_id"]},
        headers=manager_headers,
    )
    operations = call(ctx.pos, "listCheckFinancialOperations", path_params={"id": sale["check_id"]}, headers=manager_headers)
    if not any(item.get("id") == refund.get("id") and item.get("operation_type") == "refund" for item in operations):
        raise AssertionError("check financial operations did not include the recorded refund")

    check_after = call(ctx.pos, "getCheck", path_params={"id": sale["check_id"]}, headers=manager_headers)
    order_after = call(ctx.pos, "getOrder", path_params={"id": sale["order_id"]}, headers=manager_headers)
    if check_after.get("status") not in ("paid", "refunded", "voided"):
        raise AssertionError("refund mutated final check into unexpected status " + str(check_after.get("status")))
    if order_after.get("status") != "closed":
        raise AssertionError("refund mutated closed order into status " + str(order_after.get("status")))

    closed_orders = call(
        ctx.pos,
        "listClosedOrders",
        query={"shift_id": original_employee_shift["id"], "check_id": sale["check_id"], "limit": 10},
        headers=manager_headers,
    )
    if not find_by_id(closed_orders, sale["order_id"]):
        raise AssertionError("bounded closed orders read did not include the refunded order")

    result = passed_result(
        "pos_refund_after_shift_close",
        {
            "reused_existing_pairing": bool(existing_summary),
            "runtime_actor_role": session["runtime_role"],
            "runtime_actor_employee_id": runtime_employee_id,
            "manager_employee_id": manager_employee_id,
            "original_employee_shift_id": original_employee_shift.get("id"),
            "original_employee_shift_status": closed_employee_shift.get("status"),
            "original_cash_shift_id": original_cash_shift.get("id"),
            "original_cash_shift_status": closed_cash_shift.get("status"),
            "refund_employee_shift_id": refund_employee_shift.get("id"),
            "refund_cash_shift_id": refund_cash_shift.get("id"),
            "order_id": sale["order_id"],
            "precheck_id": sale["precheck_id"],
            "payment_id": sale["payment_id"],
            "check_id": sale["check_id"],
            "check_status_after_refund": check_after.get("status"),
            "order_status_after_refund": order_after.get("status"),
            "refund_operation_id": refund.get("id"),
            "refund_operation_kind": refund.get("operation_kind"),
            "refund_operation_amount": refund.get("amount"),
            "financial_operations_count": len(operations),
            "closed_orders_count": len(closed_orders),
        },
        started=started,
    )
    result["_artifacts"] = {"masterdata_summary": dict(summary)}
    return result


def prepare_pos_runtime_session(ctx, summary, client_device_id):
    manager_login = login_with_pin(ctx.pos, summary["node_device_id"], client_device_id, summary["manager_pin"])
    manager_headers = auth_headers(manager_login, summary["node_device_id"], client_device_id)
    manager_employee_id = manager_login["actor"]["employee_id"]
    if summary.get("manager_employee_id") and manager_employee_id != summary["manager_employee_id"]:
        raise AssertionError("manager PIN resolved unexpected employee_id")
    current_cash_shift = optional_call(ctx.pos, "getCurrentCashShift", headers=manager_headers)

    runtime_headers = manager_headers
    runtime_employee_id = manager_employee_id
    runtime_role = "manager"
    cashier_employee_id = summary.get("cashier_employee_id", "")
    if current_cash_shift and current_cash_shift.get("opened_by_employee_id") == cashier_employee_id and summary.get("cashier_pin"):
        cashier_login = login_with_pin(ctx.pos, summary["node_device_id"], client_device_id, summary["cashier_pin"])
        runtime_headers = auth_headers(cashier_login, summary["node_device_id"], client_device_id)
        runtime_employee_id = cashier_login["actor"]["employee_id"]
        if runtime_employee_id != cashier_employee_id:
            raise AssertionError("cashier PIN resolved unexpected employee_id")
        runtime_role = "cashier"
    elif current_cash_shift and current_cash_shift.get("opened_by_employee_id") not in ("", manager_employee_id):
        raise RuntimeError(
            "POS Edge already has an open cash shift for an employee not present in the seed summary; "
            "close/reset the local stack or provide a matching .local-masterdata-summary.json"
        )

    employee_shift = ensure_employee_shift(ctx.pos, summary, runtime_employee_id, runtime_headers)
    if current_cash_shift:
        cash_shift = current_cash_shift
        if cash_shift.get("shift_id") != employee_shift.get("id"):
            raise RuntimeError("open cash shift does not match the current employee shift for smoke actor")
    else:
        cash_shift = call(
            ctx.pos,
            "openCashShift",
            {
                "command_id": new_command_id("open-cash-shift"),
                "restaurant_id": summary["restaurant_id"],
                "opened_by_employee_id": runtime_employee_id,
                "opening_cash_amount": 0,
            },
            headers=runtime_headers,
        )
    return {
        "manager_headers": manager_headers,
        "manager_employee_id": manager_employee_id,
        "runtime_headers": runtime_headers,
        "runtime_employee_id": runtime_employee_id,
        "runtime_role": runtime_role,
        "employee_shift": employee_shift,
        "cash_shift": cash_shift,
    }


def create_pos_runtime_sale(ctx, summary, session):
    runtime_headers = session["runtime_headers"]
    employee_shift = session["employee_shift"]
    halls = call(ctx.pos, "listPOSHalls", query={"restaurant_id": summary["restaurant_id"]}, headers=runtime_headers)
    tables = call(
        ctx.pos,
        "listPOSTables",
        query={"restaurant_id": summary["restaurant_id"], "hall_id": first_hall_id(summary, halls)},
        headers=runtime_headers,
    )
    menu_items = call(ctx.pos, "listPOSMenuItems", headers=runtime_headers)
    table = choose_table(summary, tables)
    normal_item = choose_regular_menu_item(summary, menu_items)
    modifier_item, modifier = choose_modifier_menu_item(summary, menu_items)
    service_item = choose_service_menu_item(summary, menu_items)

    order = call(
        ctx.pos,
        "createOrder",
        {
            "command_id": new_command_id("create-order"),
            "restaurant_id": summary["restaurant_id"],
            "shift_id": employee_shift["id"],
            "table_id": table["id"],
            "table_name": table.get("name", ""),
            "guest_count": 1,
        },
        headers=runtime_headers,
    )
    order_id = order["id"]
    line_count = 0
    normal_line = call(
        ctx.pos,
        "addOrderLine",
        {
            "command_id": new_command_id("add-line"),
            "menu_item_id": normal_item["id"],
            "quantity": 1,
        },
        path_params={"id": order_id},
        headers=runtime_headers,
    )
    line_count += 1

    modifier_line_added = False
    if modifier_item and modifier:
        modifier_line = call(
            ctx.pos,
            "addOrderLine",
            {
                "command_id": new_command_id("add-modifier-line"),
                "menu_item_id": modifier_item["id"],
                "quantity": 1,
            },
            path_params={"id": order_id},
            headers=runtime_headers,
        )
        call(
            ctx.pos,
            "updateOrderLineModifiers",
            {
                "command_id": new_command_id("update-modifiers"),
                "selected_modifiers": [modifier],
            },
            path_params={"id": order_id, "line_id": modifier_line["id"]},
            headers=runtime_headers,
        )
        line_count += 1
        modifier_line_added = True

    service_line_added = False
    if service_item:
        call(
            ctx.pos,
            "addOrderLine",
            {
                "command_id": new_command_id("add-service-line"),
                "menu_item_id": service_item["id"],
                "quantity": 1,
            },
            path_params={"id": order_id},
            headers=runtime_headers,
        )
        line_count += 1
        service_line_added = True

    precheck = call(
        ctx.pos,
        "issuePrecheck",
        {"command_id": new_command_id("issue-precheck")},
        path_params={"id": order_id},
        headers=runtime_headers,
    )
    precheck_id = precheck["id"]
    amount = int(precheck.get("remaining_total") or precheck.get("total") or 0)
    if amount <= 0:
        raise AssertionError("precheck did not return a positive remaining amount")
    payment = call(
        ctx.pos,
        "capturePrecheckPayment",
        {
            "command_id": new_command_id("capture-payment"),
            "method": "cash",
            "amount": amount,
            "currency": precheck.get("currency_code", "RUB"),
        },
        path_params={"id": precheck_id},
        headers=runtime_headers,
    )
    closed_order = call(ctx.pos, "getOrder", path_params={"id": order_id}, headers=runtime_headers)
    check = closed_order.get("check") or find_check_in_closed_orders(ctx.pos, order_id, employee_shift, runtime_headers)
    if not check or not check.get("id"):
        raise AssertionError("final check was not created after full precheck payment")
    return {
        "order_id": order_id,
        "line_count": line_count,
        "normal_line_id": normal_line.get("id"),
        "modifier_line_added": modifier_line_added,
        "service_line_added": service_line_added,
        "precheck_id": precheck_id,
        "payment_id": payment.get("id"),
        "payment_status": payment.get("status"),
        "check_id": check["id"],
        "closed_order": closed_order,
    }


def ensure_pos_runtime_summary(ctx, **options):
    existing_summary = options.get("existing_summary")
    provisioning_status = call(ctx.pos, "getProvisioningStatus")
    if provisioning_status.get("paired"):
        return matching_existing_summary(existing_summary, provisioning_status)
    result = run_cloud_to_edge_masterdata_suite(ctx, **options)
    if result.get("status") != STATUS_PASSED:
        raise RuntimeError("cloud_to_edge_masterdata setup failed before POS cashier runtime smoke")
    return result.get("_artifacts", {}).get("masterdata_summary") or {}


def ensure_employee_shift(pos_client, summary, employee_id, headers):
    current = call(pos_client, "getCurrentEmployeeShift", headers=headers)
    if current:
        return current
    return call(
        pos_client,
        "openEmployeeShift",
        {
            "command_id": new_command_id("open-employee-shift"),
            "restaurant_id": summary["restaurant_id"],
            "opened_by_employee_id": employee_id,
        },
        headers=headers,
    )


def optional_call(client, operation_id, **kwargs):
    try:
        return call(client, operation_id, **kwargs)
    except HttpError as exc:
        if exc.status == 404:
            return None
        raise


def new_command_id(prefix):
    return prefix + "-" + uuid.uuid4().hex


def first_hall_id(summary, halls):
    if summary.get("hall_id"):
        return summary["hall_id"]
    if summary.get("hall_ids"):
        return summary["hall_ids"][0]
    hall = first_item(halls)
    if hall and hall.get("id"):
        return hall["id"]
    raise AssertionError("POS hall read model is empty")


def choose_table(summary, tables):
    for table_id in summary.get("table_ids", []):
        found = find_by_id(tables, table_id)
        if found:
            return found
    table = first_item(tables)
    if table and table.get("id"):
        return table
    raise AssertionError("POS table read model is empty")


def choose_regular_menu_item(summary, menu_items):
    candidates = preferred_menu_items(summary, menu_items)
    for item in candidates:
        if item.get("active", True) and item.get("item_type") != "service":
            return item
    raise AssertionError("POS menu read model does not contain an active regular menu item")


def choose_modifier_menu_item(summary, menu_items):
    for item in preferred_menu_items(summary, menu_items):
        if not item.get("active", True) or item.get("item_type") == "service":
            continue
        for group in item.get("modifier_groups", []) or []:
            if not group.get("active", True) or int(group.get("max_count") or 0) <= 0:
                continue
            for option in group.get("options", []) or []:
                if option.get("active", True):
                    return item, {
                        "modifier_group_id": group["id"],
                        "modifier_option_id": option["id"],
                        "quantity": 1,
                    }
    return None, None


def choose_service_menu_item(summary, menu_items):
    for item in preferred_menu_items(summary, menu_items):
        if item.get("active", True) and item.get("item_type") == "service":
            return item
    return None


def preferred_menu_items(summary, menu_items):
    items = flatten_dicts(menu_items)
    preferred_ids = set(summary.get("menu_item_ids", []))
    if preferred_ids:
        preferred = [item for item in items if item.get("id") in preferred_ids]
        if preferred:
            return preferred
    return items


def first_item(value):
    items = flatten_dicts(value)
    return items[0] if items else None


def flatten_dicts(value):
    if isinstance(value, dict):
        if value.get("id"):
            return [value]
        out = []
        for child in value.values():
            out.extend(flatten_dicts(child))
        return out
    if isinstance(value, list):
        out = []
        for child in value:
            out.extend(flatten_dicts(child))
        return out
    return []


def find_check_in_closed_orders(pos_client, order_id, shift, headers):
    closed_orders = call(
        pos_client,
        "listClosedOrders",
        query={"shift_id": shift["id"], "limit": 10},
        headers=headers,
    )
    found = find_by_id(closed_orders, order_id)
    return found.get("check") if isinstance(found, dict) else None


def find_by_id(value, expected_id):
    if isinstance(value, dict):
        if value.get("id") == expected_id:
            return value
        for child in value.values():
            found = find_by_id(child, expected_id)
            if found:
                return found
    if isinstance(value, list):
        for child in value:
            found = find_by_id(child, expected_id)
            if found:
                return found
    return None


def storage_status_details(status):
    if not isinstance(status, dict):
        return {}
    counts = status.get("counts") if isinstance(status.get("counts"), dict) else {}
    return {
        "retention_mode": status.get("retention_mode"),
        "database_size_bytes": status.get("database_size_bytes"),
        "closed_orders": counts.get("closed_orders"),
        "checks": counts.get("checks"),
        "financial_operations": counts.get("financial_operations"),
        "outbox_pending": status.get("outbox_pending"),
        "outbox_failed": status.get("outbox_failed"),
    }


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
            elif suite == "pos_cashier_runtime":
                suite_options = dict(options)
                if artifacts.get("masterdata_summary"):
                    suite_options["existing_summary"] = artifacts["masterdata_summary"]
                result = run_pos_cashier_runtime_suite(ctx, **suite_options)
            elif suite == "pos_refund_after_shift_close":
                suite_options = dict(options)
                if artifacts.get("masterdata_summary"):
                    suite_options["existing_summary"] = artifacts["masterdata_summary"]
                result = run_pos_refund_after_shift_close_suite(ctx, **suite_options)
            else:
                result = skipped_result(suite, "unknown suite", started=started)
        except Exception as exc:
            result = failed_result(suite, exc, started=started)
        suite_artifacts = result.pop("_artifacts", {})
        artifacts.update(suite_artifacts)
        results.append(result)
    return stack_result(results, artifacts=artifacts)
