#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys


SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR / "lib"))

from mhpos_http import JsonClient, normalize_base_url
from mhpos_seed import (
    auth_headers,
    login_with_pin,
    read_summary,
    verify_pos_read_model,
    verify_sync_status,
    wait_for_menu_item,
)


def parse_args():
    parser = argparse.ArgumentParser(description="Verify POS Edge read model and Cloud -> Edge sync.")
    parser.add_argument("--pos-base", default="http://localhost:8080")
    parser.add_argument("--summary", required=True)
    parser.add_argument("--client-device-id", default="python-masterdata-smoke-client")
    parser.add_argument("--expect-menu-item-id", default="")
    parser.add_argument("--wait-seconds", type=int, default=90)
    parser.add_argument("--interval-seconds", type=float, default=2)
    return parser.parse_args()


def main():
    args = parse_args()
    summary = read_summary(args.summary)
    pos_base = normalize_base_url(args.pos_base or summary.get("edge_base_url", "http://localhost:8080"))
    pos = JsonClient(pos_base)
    read_result = verify_pos_read_model(pos, summary, client_device_id=args.client_device_id)
    login = login_with_pin(pos, summary["node_device_id"], args.client_device_id, summary["manager_pin"])
    headers = auth_headers(login, summary["node_device_id"], args.client_device_id)
    sync_item = None
    if args.expect_menu_item_id:
        sync_item = wait_for_menu_item(pos, args.expect_menu_item_id, headers, args.wait_seconds, args.interval_seconds)
    result = {
        "read_model": {k: v for k, v in read_result.items() if k != "headers"},
        "expected_synced_menu_item": sync_item,
        "sync": verify_sync_status(pos, headers),
    }
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
