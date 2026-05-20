#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys

SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR / "lib"))

from mhpos_http import JsonClient, normalize_base_url
from mhpos_seed import (
    DEFAULT_REFERENCE_SEED,
    create_cloud_seed,
    generate_edge_sync_events,
    get_edge_node_device_id,
    seed_reference_data,
    stamp_summary,
)


def parse_args():
    parser = argparse.ArgumentParser(description="Interactive console for live seed management.")
    parser.add_argument("--cloud-base", default="http://localhost:8090")
    parser.add_argument("--pos-base", default="http://localhost:8080")
    parser.add_argument("--restaurant-id", default="")
    parser.add_argument("--node-device-id", default="")
    parser.add_argument("--seed-file", default="")
    parser.add_argument("--cashier-pin", default="1111")
    parser.add_argument("--manager-pin", default="2222")
    return parser.parse_args()


def prompt(text, default=""):
    value = input(f"{text}" + (f" [{default}]" if default else "") + ": ").strip()
    return value or default


def load_seed_plan(path):
    if not path:
        return json.loads(json.dumps(DEFAULT_REFERENCE_SEED))
    with open(path, "r", encoding="utf-8") as fh:
        return json.load(fh)


def main():
    args = parse_args()
    cloud = JsonClient(normalize_base_url(args.cloud_base))
    pos = JsonClient(normalize_base_url(args.pos_base))
    node_device_id = args.node_device_id or get_edge_node_device_id(pos)
    seed_plan = load_seed_plan(args.seed_file)
    context = {
        "restaurant_id": args.restaurant_id,
        "node_device_id": node_device_id,
        "seed_plan": seed_plan,
    }

    while True:
        print("\n=== MH POS live seed console ===")
        print("1) Bootstrap full demo tenant + publish")
        print("2) Publish current seed plan into existing restaurant")
        print("3) Generate edge->cloud sync batches")
        print("4) Add catalog item to in-memory seed plan")
        print("5) Save current seed plan to file")
        print("0) Exit")
        choice = input("Choose action: ").strip()

        if choice == "0":
            break
        if choice == "1":
            summary = create_cloud_seed(
                cloud,
                cashier_pin=args.cashier_pin,
                manager_pin=args.manager_pin,
                node_device_id=context["node_device_id"],
            )
            summary = stamp_summary(summary, args.cloud_base, args.pos_base)
            context["restaurant_id"] = summary["restaurant_id"]
            print(json.dumps(summary, ensure_ascii=False, indent=2))
            continue
        if choice == "2":
            restaurant_id = context["restaurant_id"] or prompt("restaurant_id")
            result = seed_reference_data(
                cloud,
                restaurant_id,
                context["seed_plan"],
                publication_node_device_id=context["node_device_id"],
                published_by="python-live-seed-console",
            )
            context["restaurant_id"] = restaurant_id
            print(json.dumps(result, ensure_ascii=False, indent=2))
            continue
        if choice == "3":
            restaurant_id = context["restaurant_id"] or prompt("restaurant_id")
            batches = int(prompt("batches", "3"))
            summary = {"restaurant_id": restaurant_id, "node_device_id": context["node_device_id"]}
            events = generate_edge_sync_events(cloud, summary, context["seed_plan"], batches=batches)
            print(json.dumps({"batches": batches, "events": events}, ensure_ascii=False, indent=2))
            continue
        if choice == "4":
            item = {
                "kind": prompt("kind", "dish"),
                "name": prompt("name", "New item"),
                "sku": prompt("sku", "SEED-NEW"),
                "base_unit": prompt("base_unit", "portion"),
                "price_minor": int(prompt("price_minor", "10000")),
            }
            context["seed_plan"].setdefault("catalog", []).append(item)
            print(json.dumps(item, ensure_ascii=False, indent=2))
            continue
        if choice == "5":
            target = prompt("file", args.seed_file or "scripts/seed-plan.local.json")
            with open(target, "w", encoding="utf-8") as fh:
                json.dump(context["seed_plan"], fh, ensure_ascii=False, indent=2)
                fh.write("\n")
            print(f"saved: {target}")
            continue

        print("Unknown action")


if __name__ == "__main__":
    main()
