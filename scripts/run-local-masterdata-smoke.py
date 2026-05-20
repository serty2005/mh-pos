#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys


SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR / "lib"))

from mhpos_http import JsonClient, normalize_base_url
from mhpos_seed import (
    create_cloud_seed,
    create_post_pairing_sync_item,
    get_edge_node_device_id,
    health_check,
    provision_pos_edge,
    stamp_summary,
    verify_pos_read_model,
    verify_sync_status,
    wait_for_menu_item,
    redacted_summary,
    write_summary,
)


def parse_args():
    parser = argparse.ArgumentParser(description="Run local Cloud master-data seed and POS sync smoke.")
    parser.add_argument("--cloud-base", default="http://localhost:8090")
    parser.add_argument("--pos-base", default="http://localhost:8080")
    parser.add_argument("--license-base", default="http://localhost:8095")
    parser.add_argument("--restaurant-name", default="")
    parser.add_argument("--cashier-pin", default="1111")
    parser.add_argument("--manager-pin", default="2222")
    parser.add_argument("--node-device-id", default="")
    parser.add_argument("--suffix", default="")
    parser.add_argument("--client-device-id", default="python-masterdata-smoke-client")
    parser.add_argument("--output", default="scripts/.local-masterdata-summary.json")
    parser.add_argument("--skip-post-pairing-sync-check", action="store_true")
    parser.add_argument("--wait-seconds", type=int, default=90)
    parser.add_argument("--interval-seconds", type=float, default=2)
    return parser.parse_args()


def main():
    args = parse_args()
    cloud_base = normalize_base_url(args.cloud_base)
    pos_base = normalize_base_url(args.pos_base)
    license_base = normalize_base_url(args.license_base)
    cloud = JsonClient(cloud_base)
    pos = JsonClient(pos_base)
    license_client = JsonClient(license_base)

    health = {
        "cloud": health_check(cloud),
        "pos": health_check(pos),
        "license": health_check(license_client),
    }
    node_device_id = args.node_device_id or get_edge_node_device_id(pos)
    summary = create_cloud_seed(
        cloud,
        restaurant_name=args.restaurant_name,
        cashier_pin=args.cashier_pin,
        manager_pin=args.manager_pin,
        node_device_id=node_device_id,
        suffix=args.suffix,
    )
    provisioning = provision_pos_edge(cloud, pos, cloud_base, summary["restaurant_id"], node_device_id)
    summary.update(provisioning)
    summary = stamp_summary(summary, cloud_base, pos_base)
    initial_read = verify_pos_read_model(pos, summary, client_device_id=args.client_device_id)
    post_pairing = None
    synced_item = None
    if not args.skip_post_pairing_sync_check:
        post_pairing = create_post_pairing_sync_item(cloud, summary, suffix=summary.get("suffix", ""))
        synced_item = wait_for_menu_item(
            pos,
            post_pairing["menu_item_id"],
            initial_read["headers"],
            timeout_seconds=args.wait_seconds,
            interval_seconds=args.interval_seconds,
        )
        summary["post_pairing_sync_menu_item_id"] = post_pairing["menu_item_id"]
    sync_status = verify_sync_status(pos, initial_read["headers"])
    result = {
        "health": health,
        "summary": redacted_summary(summary),
        "initial_read_model": {k: v for k, v in initial_read.items() if k != "headers"},
        "post_pairing_sync": post_pairing,
        "synced_item": synced_item,
        "sync": sync_status,
    }
    write_summary(args.output, summary)
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
