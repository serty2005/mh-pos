#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys


SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR / "lib"))

from mhpos_http import JsonClient, normalize_base_url
from mhpos_seed import create_cloud_seed, get_edge_node_device_id, redacted_summary, stamp_summary, write_summary


def parse_args():
    parser = argparse.ArgumentParser(description="Seed Cloud master data through Cloud API.")
    parser.add_argument("--cloud-base", default="http://localhost:8090")
    parser.add_argument("--pos-base", default="http://localhost:8080")
    parser.add_argument("--restaurant-name", default="")
    parser.add_argument("--node-device-id", default="")
    parser.add_argument("--cashier-pin", default="1111")
    parser.add_argument("--manager-pin", default="2222")
    parser.add_argument("--suffix", default="")
    parser.add_argument("--output", default="")
    parser.add_argument("--skip-pos-node-detect", action="store_true")
    return parser.parse_args()


def main():
    args = parse_args()
    cloud_base = normalize_base_url(args.cloud_base)
    pos_base = normalize_base_url(args.pos_base)
    cloud = JsonClient(cloud_base)
    pos = JsonClient(pos_base)
    node_device_id = args.node_device_id
    if not node_device_id and not args.skip_pos_node_detect:
        node_device_id = get_edge_node_device_id(pos)
    summary = create_cloud_seed(
        cloud,
        restaurant_name=args.restaurant_name,
        cashier_pin=args.cashier_pin,
        manager_pin=args.manager_pin,
        node_device_id=node_device_id,
        suffix=args.suffix,
    )
    summary = stamp_summary(summary, cloud_base, pos_base)
    write_summary(args.output, summary)
    print(json.dumps(redacted_summary(summary), ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
