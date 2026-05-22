#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys


SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR / "lib"))

from mhpos_http import JsonClient, normalize_base_url
from mhpos_seed import read_summary, write_summary
from mhpos_stack import StackContext, parse_suites, run_selected_suites


def parse_args(argv=None):
    parser = argparse.ArgumentParser(description="Run local full-stack smoke suites for Cloud, POS Edge and License Server.")
    parser.add_argument("--cloud-base", default="http://localhost:8090")
    parser.add_argument("--pos-base", default="http://localhost:8080")
    parser.add_argument("--license-base", default="http://localhost:8095")
    parser.add_argument("--suite", action="append", default=[], help="Suite name, comma-separated names, or all.")
    parser.add_argument("--restaurant-name", default="")
    parser.add_argument("--cashier-pin", default="1111")
    parser.add_argument("--manager-pin", default="2222")
    parser.add_argument("--node-device-id", default="")
    parser.add_argument("--suffix", default="")
    parser.add_argument("--client-device-id", default="python-stack-smoke-client")
    parser.add_argument("--output", default="scripts/.local-masterdata-summary.json")
    parser.add_argument("--json-output", default="")
    parser.add_argument("--skip-post-pairing-sync-check", action="store_true")
    parser.add_argument("--wait-seconds", type=int, default=90)
    parser.add_argument("--interval-seconds", type=float, default=2)
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    cloud_base = normalize_base_url(args.cloud_base)
    pos_base = normalize_base_url(args.pos_base)
    license_base = normalize_base_url(args.license_base)
    ctx = StackContext(
        cloud=JsonClient(cloud_base),
        pos=JsonClient(pos_base),
        license=JsonClient(license_base),
        cloud_base_url=cloud_base,
        pos_base_url=pos_base,
        license_base_url=license_base,
    )
    suites = parse_suites(args.suite)
    existing_summary = None
    output_path = pathlib.Path(args.output) if args.output else None
    if output_path and output_path.exists():
        existing_summary = read_summary(output_path)
    result = run_selected_suites(
        ctx,
        suites,
        restaurant_name=args.restaurant_name,
        cashier_pin=args.cashier_pin,
        manager_pin=args.manager_pin,
        node_device_id=args.node_device_id,
        suffix=args.suffix,
        client_device_id=args.client_device_id,
        skip_post_pairing_sync_check=args.skip_post_pairing_sync_check,
        wait_seconds=args.wait_seconds,
        interval_seconds=args.interval_seconds,
        existing_summary=existing_summary,
    )
    artifacts = result.pop("_artifacts", {})
    summary = artifacts.get("masterdata_summary")
    if summary and args.output:
        write_summary(args.output, summary)
    if args.json_output:
        write_summary(args.json_output, result)
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0 if result["status"] != "failed" else 1


# Расширение smoke для Storage Retention Lifecycle (destructive apply + VACUUM):
# В mhpos_stack.py в run_pos_cashier_runtime_suite добавлены проверки:
# 1. /storage/archive/apply-readiness -> ready_for_destructive_apply=true после export+verify (для старого cutoff)
# 2. /storage/archive/apply-plan -> runtime_rows_deleted=true, result_mode=destructive_apply без ошибок
# 3. /orders/closed продолжает возвращать актуальные записи (не удалены)
# 4. /storage/archive/read-plan и /lookup возвращают превью из JSONL-архива
# (использован старый cutoff 2020-01-01 чтобы не затрагивать runtime данные smoke)


if __name__ == "__main__":
    raise SystemExit(main())
