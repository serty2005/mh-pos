#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys


SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR / "lib"))

from mhpos_http import JsonClient, normalize_base_url
from mhpos_seed import get_edge_node_device_id, provision_pos_edge, read_summary, redacted_summary, write_summary


def parse_args():
    parser = argparse.ArgumentParser(description="Provision POS Edge through Cloud/License APIs.")
    parser.add_argument("--cloud-base", default="http://localhost:8090")
    parser.add_argument("--pos-base", default="http://localhost:8080")
    parser.add_argument("--summary", default="")
    parser.add_argument("--restaurant-id", default="")
    parser.add_argument("--node-device-id", default="")
    parser.add_argument("--display-name", default="POS Terminal 1")
    parser.add_argument("--output", default="")
    parser.add_argument("--no-assignment-fallback", action="store_true")
    return parser.parse_args()


def main():
    args = parse_args()
    summary = read_summary(args.summary) if args.summary else {}
    cloud_base = normalize_base_url(args.cloud_base or summary.get("cloud_base_url", "http://localhost:8090"))
    pos_base = normalize_base_url(args.pos_base or summary.get("edge_base_url", "http://localhost:8080"))
    cloud = JsonClient(cloud_base)
    pos = JsonClient(pos_base)
    restaurant_id = args.restaurant_id or summary.get("restaurant_id", "")
    node_device_id = args.node_device_id or summary.get("node_device_id", "") or get_edge_node_device_id(pos)
    if not restaurant_id:
        raise SystemExit("restaurant_id is required")
    result = provision_pos_edge(
        cloud,
        pos,
        cloud_base,
        restaurant_id,
        node_device_id,
        display_name=args.display_name,
        allow_assignment_fallback=not args.no_assignment_fallback,
    )
    summary.update(result)
    summary["restaurant_id"] = restaurant_id
    summary["node_device_id"] = node_device_id
    write_summary(args.output or args.summary, summary)
    print(json.dumps(redacted_summary(summary), ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
