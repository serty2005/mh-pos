import json
import time
import uuid

from mhpos_contract import load_default_contract
from mhpos_http import wait_until


CONTRACT = load_default_contract()

CASHIER_PERMISSIONS = [
    "pos.employee_shift.open",
    "pos.employee_shift.close",
    "pos.employee_shift.view_current",
    "pos.employee_shift.recent",
    "pos.cash_session.open",
    "pos.cash_session.view_current",
    "pos.catalog.view",
    "pos.floor.view",
    "pos.menu.view",
    "pos.order.create",
    "pos.order.view",
    "pos.order.add_line",
    "pos.order.change_quantity",
    "pos.order.void_line",
    "pos.order.close",
    "pos.precheck.issue",
    "pos.precheck.view",
    "pos.precheck.reprint",
    "pos.payment.cash",
    "pos.payment.card.manual",
    "pos.check.view",
]

MANAGER_PERMISSIONS = CASHIER_PERMISSIONS + [
    "pos.cash_session.close",
    "pos.cash_drawer.record_event",
    "pos.precheck.cancel.request",
    "pos.precheck.cancel",
    "pos.payment.other",
    "pos.payment.refund",
    "pos.check.reprint",
    "pos.sync.view",
    "pos.sync.retry_failed",
]


def command_suffix():
    return uuid.uuid4().hex[:8]


def permissions_json(items):
    return json.dumps({item: True for item in sorted(set(items))}, separators=(",", ":"))


def call(client, operation_id, body=None, path_params=None, query=None, headers=None, expected_status=None):
    request = CONTRACT.build_request(operation_id, path_params=path_params, query=query, body=body)
    statuses = expected_status or request["expected_status"]
    if hasattr(client, "request"):
        return client.request(
            request["method"],
            request["path"],
            body,
            headers,
            statuses,
            api_prefix=request["api_prefix"],
        )
    if not request["api_prefix"]:
        return client.root_get(request["path"], headers=headers, expected_status=statuses)
    if request["method"] == "GET":
        return client.get(request["path"], headers=headers, expected_status=statuses)
    if request["method"] == "POST":
        return client.post(request["path"], body or {}, headers=headers, expected_status=statuses)
    if request["method"] == "PATCH":
        return client.patch(request["path"], body or {}, headers=headers, expected_status=statuses)
    if request["method"] == "PUT":
        return client.put(request["path"], body or {}, headers=headers, expected_status=statuses)
    raise ValueError(f"unsupported HTTP method {request['method']} for {operation_id}")


def health_check(raw_client):
    return call(raw_client, "health")


def create_cloud_seed(
    client,
    restaurant_name="",
    cashier_pin="1111",
    manager_pin="2222",
    node_device_id="",
    suffix="",
    business_day_boundary_local_time="04:00",
):
    suffix = suffix or command_suffix()
    restaurant = call(
        client,
        "createRestaurant",
        {
            "name": restaurant_name or f"Local Demo Bistro {suffix}",
            "timezone": "Europe/Moscow",
            "currency": "RUB",
            "business_day_mode": "standard",
            "business_day_boundary_local_time": business_day_boundary_local_time,
        },
    )
    restaurant_id = restaurant["id"]
    cashier_role = call(
        client,
        "createRole",
        {
            "restaurant_id": restaurant_id,
            "name": f"cashier-{suffix}",
            "permissions_json": permissions_json(CASHIER_PERMISSIONS),
        },
    )
    manager_role = call(
        client,
        "createRole",
        {
            "restaurant_id": restaurant_id,
            "name": f"manager-{suffix}",
            "permissions_json": permissions_json(MANAGER_PERMISSIONS),
        },
    )
    cashier = call(
        client,
        "createEmployee",
        {"restaurant_id": restaurant_id, "role_id": cashier_role["id"], "name": "Demo Cashier", "pin": cashier_pin},
    )
    manager = call(
        client,
        "createEmployee",
        {"restaurant_id": restaurant_id, "role_id": manager_role["id"], "name": "Demo Manager", "pin": manager_pin},
    )
    hall = call(client, "createHall", {"restaurant_id": restaurant_id, "name": "Main Hall"})
    table = call(
        client,
        "createTable",
        {"restaurant_id": restaurant_id, "hall_id": hall["id"], "name": "T1", "seats": 2},
    )

    catalog_tea = create_catalog_item(client, restaurant_id, "dish", f"Demo Tea {suffix}", f"DEMO-TEA-{suffix}", "portion")
    catalog_soup = create_catalog_item(client, restaurant_id, "dish", f"Demo Soup {suffix}", f"DEMO-SOUP-{suffix}", "portion")
    catalog_service = create_catalog_item(client, restaurant_id, "service", f"Demo Service {suffix}", f"DEMO-SERVICE-{suffix}", "service")

    menu_tea = create_menu_item(client, restaurant_id, catalog_tea["id"], "Demo Tea", 15000)
    menu_soup = create_menu_item(client, restaurant_id, catalog_soup["id"], "Demo Soup", 25000)
    menu_service = create_menu_item(client, restaurant_id, catalog_service["id"], "Demo Service", 5000)

    modifier_group = call(
        client,
        "createModifierGroup",
        {"restaurant_id": restaurant_id, "name": "Demo Add-ons", "required": False, "min_count": 0, "max_count": 2},
    )
    modifier_option = call(
        client,
        "createModifierOption",
        {
            "restaurant_id": restaurant_id,
            "modifier_group_id": modifier_group["id"],
            "name": "Lemon",
            "price_minor": 3000,
        },
    )
    modifier_binding = call(
        client,
        "createModifierBinding",
        {
            "restaurant_id": restaurant_id,
            "modifier_group_id": modifier_group["id"],
            "target_type": "menu_item",
            "target_id": menu_tea["id"],
            "sort_order": 1,
        },
    )
    publication = publish_master_data(client, restaurant_id, node_device_id, "python-seed")
    return {
        "restaurant_id": restaurant_id,
        "node_device_id": node_device_id,
        "cashier_pin": cashier_pin,
        "manager_pin": manager_pin,
        "cashier_employee_id": cashier["id"],
        "manager_employee_id": manager["id"],
        "hall_id": hall["id"],
        "table_ids": [table["id"]],
        "catalog_item_ids": [catalog_tea["id"], catalog_soup["id"], catalog_service["id"]],
        "menu_item_ids": [menu_tea["id"], menu_soup["id"], menu_service["id"]],
        "modifier_group_id": modifier_group["id"],
        "modifier_option_id": modifier_option["id"],
        "modifier_binding_id": modifier_binding["id"],
        "publication_id": publication["id"],
        "suffix": suffix,
        "business_day_boundary_local_time": business_day_boundary_local_time,
    }


def create_catalog_item(client, restaurant_id, kind, name, sku, base_unit):
    return call(
        client,
        "createCatalogItem",
        {"restaurant_id": restaurant_id, "kind": kind, "name": name, "sku": sku, "base_unit": base_unit},
    )


def create_menu_item(client, restaurant_id, catalog_item_id, name, price):
    return call(
        client,
        "createMenuItem",
        {
            "restaurant_id": restaurant_id,
            "catalog_item_id": catalog_item_id,
            "name": name,
            "price": price,
            "currency": "RUB",
            "availability_json": "{}",
        },
    )


def create_recipe_item(client, restaurant_id, owner_catalog_item_id, component_catalog_item_id, quantity=1, unit="g", loss_percent=0):
    return call(
        client,
        "createRecipeItem",
        {
            "restaurant_id": restaurant_id,
            "recipe_owner_catalog_item_id": owner_catalog_item_id,
            "component_catalog_item_id": component_catalog_item_id,
            "quantity": int(quantity),
            "unit": unit,
            "loss_percent": int(loss_percent),
        },
    )


def create_stop_list_entry(client, restaurant_id, catalog_item_id, available_quantity=0, reason="stack smoke stop-list"):
    return call(
        client,
        "createStopListEntry",
        {
            "restaurant_id": restaurant_id,
            "catalog_item_id": catalog_item_id,
            "available_quantity": available_quantity,
            "reason": reason,
            "active": True,
        },
    )


def publish_master_data(client, restaurant_id, node_device_id="", published_by="python-masterdata-seed"):
    body = {"restaurant_id": restaurant_id, "published_by": published_by}
    if node_device_id:
        body["node_device_id"] = node_device_id
    return call(client, "publishMasterData", body)


def get_edge_node_device_id(pos_client):
    status = call(pos_client, "getProvisioningStatus")
    node_device_id = status.get("node_device_id", "")
    if not node_device_id:
        raise RuntimeError("POS Edge did not return node_device_id")
    return node_device_id


def provision_via_license(cloud_client, pos_client, restaurant_id, node_device_id, display_name="POS Terminal 1"):
    pairing = call(
        cloud_client,
        "generatePairingCode",
        {"node_device_id": node_device_id, "display_name": display_name, "expires_in_minutes": 30},
        path_params={"restaurant_id": restaurant_id},
    )
    paired = call(pos_client, "pairViaLicense", {"pairing_code": pairing["pairing_code"]})
    return {"pairing_code": pairing["pairing_code"], "pairing_status": paired}


def provision_pos_edge(cloud_client, pos_client, cloud_base_url, restaurant_id, node_device_id, display_name="POS Terminal 1", allow_assignment_fallback=True):
    try:
        result = provision_via_license(cloud_client, pos_client, restaurant_id, node_device_id, display_name)
        result["provisioning_mode"] = "license_code"
        return result
    except Exception:
        if not allow_assignment_fallback:
            raise
    try:
        call(
            pos_client,
            "registerCloudProvisioning",
            {"cloud_url": cloud_base_url.rstrip("/"), "display_name": display_name, "app_version": "local-python-smoke"},
        )
    except Exception:
        pass
    assignment = call(
        cloud_client,
        "assignDevice",
        path_params={"restaurant_id": restaurant_id, "node_device_id": node_device_id},
    )

    def check():
        status = call(pos_client, "getProvisioningStatus")
        return status if status.get("paired") else None

    paired = wait_until(check, timeout_seconds=60, interval_seconds=1)
    return {"pairing_code": f"Cloud-approved:{node_device_id}", "assignment": assignment, "pairing_status": paired, "provisioning_mode": "cloud_assignment"}


def auth_headers(login, node_device_id, client_device_id):
    return {
        "X-Node-Device-ID": node_device_id,
        "X-Client-Device-ID": client_device_id,
        "X-Session-ID": login["session"]["id"],
        "X-Actor-Employee-ID": login["actor"]["employee_id"],
    }


def login_with_pin(pos_client, node_device_id, client_device_id, pin):
    return call(
        pos_client,
        "pinLogin",
        {"node_device_id": node_device_id, "client_device_id": client_device_id, "pin": pin},
    )


def verify_pos_read_model(pos_client, summary, client_device_id="python-masterdata-smoke-client"):
    login = login_with_pin(pos_client, summary["node_device_id"], client_device_id, summary["manager_pin"])
    expected_manager = summary.get("manager_employee_id")
    if expected_manager and login["actor"]["employee_id"] != expected_manager:
        raise AssertionError("manager PIN resolved unexpected employee_id")
    headers = auth_headers(login, summary["node_device_id"], client_device_id)
    halls = call(pos_client, "listPOSHalls", query={"restaurant_id": summary["restaurant_id"]}, headers=headers)
    tables = call(
        pos_client,
        "listPOSTables",
        query={"restaurant_id": summary["restaurant_id"], "hall_id": summary["hall_id"]},
        headers=headers,
    )
    menu_items = call(pos_client, "listPOSMenuItems", headers=headers)
    assert_contains_id(halls, summary["hall_id"], "Cloud-created hall is not visible on POS Edge")
    for table_id in summary.get("table_ids", []):
        assert_contains_id(tables, table_id, "Cloud-created table is not visible on POS Edge")
    for menu_item_id in summary.get("menu_item_ids", []):
        assert_contains_id(menu_items, menu_item_id, "Cloud-created menu item is not visible on POS Edge")
    assert_json_contains(menu_items, summary["modifier_option_id"], "Cloud-created modifier option is not visible on POS Edge")
    return {
        "manager_employee_id": login["actor"]["employee_id"],
        "menu_items_checked": len(summary.get("menu_item_ids", [])),
        "headers": headers,
    }


def create_post_pairing_sync_item(cloud_client, summary, suffix=""):
    suffix = suffix or command_suffix()
    restaurant_id = summary["restaurant_id"]
    catalog = create_catalog_item(cloud_client, restaurant_id, "dish", f"Synced Dessert {suffix}", f"SYNC-DESSERT-{suffix}", "portion")
    menu = create_menu_item(cloud_client, restaurant_id, catalog["id"], f"Synced Dessert {suffix}", 9900)
    publication = publish_master_data(cloud_client, restaurant_id, summary.get("node_device_id", ""), "python-sync-smoke")
    return {"catalog_item_id": catalog["id"], "menu_item_id": menu["id"], "publication_id": publication["id"]}


def wait_for_menu_item(pos_client, menu_item_id, headers, timeout_seconds=90, interval_seconds=2):
    def check():
        items = call(pos_client, "listPOSMenuItems", headers=headers)
        return find_by_id(items, menu_item_id)

    return wait_until(check, timeout_seconds, interval_seconds)


def verify_sync_status(pos_client, headers):
    return {
        "sync_status": call(pos_client, "getSyncStatus", headers=headers),
        "outbox": call(pos_client, "listSyncOutbox", query={"limit": 5}, headers=headers),
        "local_events": call(pos_client, "listSyncLocalEvents", query={"limit": 5}, headers=headers),
    }


def assert_contains_id(items, expected_id, message):
    if not find_by_id(items, expected_id):
        raise AssertionError(message)


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


def assert_json_contains(value, needle, message):
    if needle not in json.dumps(value, ensure_ascii=False):
        raise AssertionError(message)


def write_summary(path, summary):
    if not path:
        return
    with open(path, "w", encoding="utf-8") as fh:
        json.dump(summary, fh, ensure_ascii=False, indent=2)
        fh.write("\n")


def read_summary(path):
    with open(path, "r", encoding="utf-8") as fh:
        return json.load(fh)


def redacted_summary(summary):
    out = dict(summary)
    for key in ("cashier_pin", "manager_pin", "pairing_code"):
        if out.get(key):
            out[key] = "<redacted>"
    return out


def stamp_summary(summary, cloud_base_url, pos_base_url):
    out = dict(summary)
    out["cloud_base_url"] = cloud_base_url
    out["edge_base_url"] = pos_base_url
    out["generated_at_unix"] = int(time.time())
    return out


DEFAULT_REFERENCE_SEED = {
    "catalog": [
        {"kind": "dish", "name": "Espresso", "sku": "SEED-ESPRESSO", "base_unit": "cup", "price_minor": 12900},
        {"kind": "dish", "name": "Cappuccino", "sku": "SEED-CAPPUCCINO", "base_unit": "cup", "price_minor": 15900},
        {"kind": "dish", "name": "Tom Yum", "sku": "SEED-TOM-YUM", "base_unit": "portion", "price_minor": 34900},
        {"kind": "service", "name": "Service fee", "sku": "SEED-SERVICE", "base_unit": "service", "price_minor": 5000},
    ],
    "halls": [
        {"name": "Main Hall", "tables": [{"name": "T1", "seats": 2}, {"name": "T2", "seats": 4}]},
        {"name": "Patio", "tables": [{"name": "P1", "seats": 2}]},
    ],
    "modifier_groups": [
        {
            "name": "Milk options",
            "required": False,
            "min_count": 0,
            "max_count": 1,
            "options": [
                {"name": "Oat milk", "price_minor": 2500},
                {"name": "Lactose free", "price_minor": 2000},
            ],
            "bind_to_sku": ["SEED-ESPRESSO", "SEED-CAPPUCCINO"],
        }
    ],
}


def seed_reference_data(client, restaurant_id, seed_plan, publication_node_device_id="", published_by="python-seed-live"):
    """Создает master-data справочники из плана и публикует их одним событием."""
    catalog_ids_by_sku = {}
    menu_item_ids = []
    hall_ids = []
    table_ids = []
    modifier_group_ids = []
    modifier_option_ids = []
    modifier_binding_ids = []

    for hall in seed_plan.get("halls", []):
        hall_created = call(client, "createHall", {"restaurant_id": restaurant_id, "name": hall["name"]})
        hall_ids.append(hall_created["id"])
        for table in hall.get("tables", []):
            table_created = call(
                client,
                "createTable",
                {
                    "restaurant_id": restaurant_id,
                    "hall_id": hall_created["id"],
                    "name": table["name"],
                    "seats": int(table.get("seats", 2)),
                },
            )
            table_ids.append(table_created["id"])

    for item in seed_plan.get("catalog", []):
        catalog_item = create_catalog_item(
            client,
            restaurant_id,
            item.get("kind", "dish"),
            item["name"],
            item["sku"],
            item.get("base_unit", "portion"),
        )
        catalog_ids_by_sku[item["sku"]] = catalog_item["id"]
        menu_item = create_menu_item(
            client,
            restaurant_id,
            catalog_item["id"],
            item.get("menu_name", item["name"]),
            int(item.get("price_minor", 10000)),
        )
        menu_item_ids.append(menu_item["id"])

    for group in seed_plan.get("modifier_groups", []):
        group_created = call(
            client,
            "createModifierGroup",
            {
                "restaurant_id": restaurant_id,
                "name": group["name"],
                "required": bool(group.get("required", False)),
                "min_count": int(group.get("min_count", 0)),
                "max_count": int(group.get("max_count", 1)),
            },
        )
        modifier_group_ids.append(group_created["id"])
        for option in group.get("options", []):
            option_created = call(
                client,
                "createModifierOption",
                {
                    "restaurant_id": restaurant_id,
                    "modifier_group_id": group_created["id"],
                    "name": option["name"],
                    "price_minor": int(option.get("price_minor", 0)),
                },
            )
            modifier_option_ids.append(option_created["id"])
        for idx, sku in enumerate(group.get("bind_to_sku", []), start=1):
            if sku not in catalog_ids_by_sku:
                continue
            # По контракту binding создается к menu_item, поэтому используем индекс создания catalog/menu пары.
            menu_target_id = menu_item_ids[list(catalog_ids_by_sku.keys()).index(sku)]
            binding = call(
                client,
                "createModifierBinding",
                {
                    "restaurant_id": restaurant_id,
                    "modifier_group_id": group_created["id"],
                    "target_type": "menu_item",
                    "target_id": menu_target_id,
                    "sort_order": idx,
                },
            )
            modifier_binding_ids.append(binding["id"])

    publication = publish_master_data(
        client,
        restaurant_id,
        publication_node_device_id,
        published_by=published_by,
    )
    return {
        "catalog_item_ids": list(catalog_ids_by_sku.values()),
        "menu_item_ids": menu_item_ids,
        "hall_ids": hall_ids,
        "table_ids": table_ids,
        "modifier_group_ids": modifier_group_ids,
        "modifier_option_ids": modifier_option_ids,
        "modifier_binding_ids": modifier_binding_ids,
        "publication_id": publication["id"],
    }


def generate_edge_sync_events(cloud_client, summary, seed_plan=None, batches=1, published_by_prefix="python-edge-event-generator"):
    """Генерирует публикации master-data, чтобы в Edge появились outbox/local events для sync тестов."""
    plan = seed_plan or DEFAULT_REFERENCE_SEED
    events = []
    for batch in range(1, max(1, int(batches)) + 1):
        batch_plan = json.loads(json.dumps(plan))
        suffix = command_suffix()
        sku_rewrite_map = {}
        for item in batch_plan.get("catalog", []):
            original_sku = item["sku"]
            rewritten_sku = f"{original_sku}-{batch}-{suffix}"
            item["sku"] = rewritten_sku
            item["name"] = f"{item['name']} {batch}-{suffix}"
            sku_rewrite_map[original_sku] = rewritten_sku
        for group in batch_plan.get("modifier_groups", []):
            rewritten_binds = []
            for sku in group.get("bind_to_sku", []):
                rewritten_binds.append(sku_rewrite_map.get(sku, sku))
            group["bind_to_sku"] = rewritten_binds
        generated = seed_reference_data(
            cloud_client,
            summary["restaurant_id"],
            batch_plan,
            publication_node_device_id=summary.get("node_device_id", ""),
            published_by=f"{published_by_prefix}-{batch}",
        )
        events.append(generated)
    return events
