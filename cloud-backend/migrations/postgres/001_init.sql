-- Чистая инициализация PostgreSQL-схемы Cloud-backend.
-- Первая и единственная миграция создаёт БД с нуля.
-- Следующая нумерованная миграция (002_*.sql) — после первого боевого внедрения.

CREATE TABLE public.cloud_catalog_folders (
    id text NOT NULL,
    restaurant_id text DEFAULT ''::text NOT NULL,
    parent_id text,
    name text NOT NULL,
    sort_order bigint DEFAULT 0 NOT NULL,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_catalog_folders_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_catalog_folders_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_catalog_folders_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_catalog_folders_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_catalog_folders_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.cloud_catalog_folders(id)
);

CREATE TABLE public.cloud_catalog_folder_parameters (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    folder_id text NOT NULL,
    parameter_key text NOT NULL,
    value_type text NOT NULL,
    value_json jsonb NOT NULL,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_catalog_folder_parameters_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_catalog_folder_parameters_parameter_key_check CHECK ((parameter_key <> ''::text)),
    CONSTRAINT cloud_catalog_folder_parameters_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_catalog_folder_parameters_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_catalog_folder_parameters_value_type_check CHECK ((value_type <> ''::text)),
    CONSTRAINT cloud_catalog_folder_parameters_folder_id_parameter_key_key UNIQUE (folder_id, parameter_key),
    CONSTRAINT cloud_catalog_folder_parameters_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_catalog_folder_parameters_folder_id_fkey FOREIGN KEY (folder_id) REFERENCES public.cloud_catalog_folders(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_catalog_items (
    id text NOT NULL,
    restaurant_id text DEFAULT ''::text NOT NULL,
    kind text NOT NULL,
    folder_id text,
    name text NOT NULL,
    sku text NOT NULL,
    base_unit text NOT NULL,
    kitchen_type text,
    accounting_category text,
    status text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    qr_confirmation_enabled boolean DEFAULT false NOT NULL,
    single_unit_per_line boolean DEFAULT false NOT NULL,
    validity_mode text,
    validity_expires_at timestamp with time zone,
    CONSTRAINT cloud_catalog_items_base_unit_check CHECK ((base_unit <> ''::text)),
    CONSTRAINT cloud_catalog_items_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_catalog_items_kind_check CHECK (kind IN ('dish','good','semi_finished','service')),
    CONSTRAINT cloud_catalog_items_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_catalog_items_sku_check CHECK ((sku <> ''::text)),
    CONSTRAINT cloud_catalog_items_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_catalog_items_validity_mode_check CHECK ((validity_mode = ANY (ARRAY['cash_session'::text, 'business_date'::text, 'absolute_date'::text]))),
    CONSTRAINT cloud_catalog_items_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_catalog_items_sku_key UNIQUE (sku)
);

CREATE TABLE public.cloud_catalog_suggestions (
    id text NOT NULL,
    suggestion_id text NOT NULL,
    restaurant_id text NOT NULL,
    catalog_item_id text,
    proposal_group_id text,
    action text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    status text NOT NULL,
    review_comment text DEFAULT ''::text NOT NULL,
    reviewed_by_employee_id text DEFAULT ''::text NOT NULL,
    reviewed_at timestamp with time zone,
    assigned_to_employee_id text DEFAULT ''::text NOT NULL,
    assigned_by_employee_id text DEFAULT ''::text NOT NULL,
    assigned_at timestamp with time zone,
    assignment_note text DEFAULT ''::text NOT NULL,
    applied_catalog_item_id text DEFAULT ''::text NOT NULL,
    source_event_id text DEFAULT ''::text NOT NULL,
    suggested_at timestamp with time zone NOT NULL,
    cloud_received_at timestamp with time zone NOT NULL,
    payload_json jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_catalog_suggestions_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'changes_requested'::text]))),
    CONSTRAINT cloud_catalog_suggestions_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_catalog_suggestions_suggestion_id_key UNIQUE (suggestion_id)
);

CREATE TABLE public.cloud_catalog_tags (
    id text NOT NULL,
    restaurant_id text DEFAULT ''::text NOT NULL,
    name text NOT NULL,
    code text NOT NULL,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_catalog_tags_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_catalog_tags_code_check CHECK ((code <> ''::text)),
    CONSTRAINT cloud_catalog_tags_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_catalog_tags_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_catalog_tags_code_key UNIQUE (code),
    CONSTRAINT cloud_catalog_tags_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_catalog_item_tags (
    restaurant_id text DEFAULT ''::text NOT NULL,
    catalog_item_id text NOT NULL,
    tag_id text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_catalog_item_tags_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_catalog_item_tags_pkey PRIMARY KEY (catalog_item_id, tag_id),
    CONSTRAINT cloud_catalog_item_tags_catalog_item_id_fkey FOREIGN KEY (catalog_item_id) REFERENCES public.cloud_catalog_items(id) ON DELETE CASCADE,
    CONSTRAINT cloud_catalog_item_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.cloud_catalog_tags(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_categories (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    sort_order bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_categories_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_categories_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_categories_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_categories_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_currency_reference (
    currency_code integer NOT NULL,
    currency_alpha_code text NOT NULL,
    minor_unit smallint NOT NULL,
    currency_iso_name text NOT NULL,
    currency_symbol text NOT NULL,
    curr_basic_name text NOT NULL,
    curr_add_name text NOT NULL,
    show_add boolean DEFAULT true NOT NULL,
    show_currency_basic_name boolean DEFAULT true NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cloud_currency_reference_curr_add_name_check CHECK ((curr_add_name <> ''::text)),
    CONSTRAINT cloud_currency_reference_curr_basic_name_check CHECK ((curr_basic_name <> ''::text)),
    CONSTRAINT cloud_currency_reference_currency_alpha_code_check CHECK ((currency_alpha_code ~ '^[A-Z]{3}$'::text)),
    CONSTRAINT cloud_currency_reference_currency_code_check CHECK ((currency_code > 0)),
    CONSTRAINT cloud_currency_reference_currency_iso_name_check CHECK ((currency_iso_name <> ''::text)),
    CONSTRAINT cloud_currency_reference_currency_symbol_check CHECK ((currency_symbol <> ''::text)),
    CONSTRAINT cloud_currency_reference_minor_unit_check CHECK (((minor_unit >= 0) AND (minor_unit <= 4))),
    CONSTRAINT cloud_currency_reference_currency_alpha_code_key UNIQUE (currency_alpha_code),
    CONSTRAINT cloud_currency_reference_pkey PRIMARY KEY (currency_code)
);

CREATE TABLE public.cloud_dishes (
    catalog_item_id text NOT NULL,
    restaurant_id text DEFAULT ''::text NOT NULL,
    recipe_policy text DEFAULT 'none'::text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_dishes_recipe_policy_check CHECK ((recipe_policy = ANY (ARRAY['none'::text, 'optional'::text, 'required'::text]))),
    CONSTRAINT cloud_dishes_pkey PRIMARY KEY (catalog_item_id),
    CONSTRAINT cloud_dishes_catalog_item_id_fkey FOREIGN KEY (catalog_item_id) REFERENCES public.cloud_catalog_items(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_edge_event_receipts (
    id text NOT NULL,
    idempotency_key text NOT NULL,
    restaurant_id text NOT NULL,
    device_id text NOT NULL,
    command_id text NOT NULL,
    event_id text NOT NULL,
    edge_event_id text NOT NULL,
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    envelope_version text NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    cloud_received_at timestamp with time zone NOT NULL,
    raw_payload_sha256_hex text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cloud_edge_event_receipts_aggregate_id_check CHECK ((aggregate_id <> ''::text)),
    CONSTRAINT cloud_edge_event_receipts_aggregate_type_check CHECK ((aggregate_type <> ''::text)),
    CONSTRAINT cloud_edge_event_receipts_command_id_check CHECK ((command_id <> ''::text)),
    CONSTRAINT cloud_edge_event_receipts_device_id_check CHECK ((device_id <> ''::text)),
    CONSTRAINT cloud_edge_event_receipts_edge_event_id_check CHECK ((edge_event_id <> ''::text)),
    CONSTRAINT cloud_edge_event_receipts_envelope_version_check CHECK ((envelope_version = '1'::text)),
    CONSTRAINT cloud_edge_event_receipts_event_id_check CHECK ((event_id <> ''::text)),
    CONSTRAINT cloud_edge_event_receipts_event_type_check CHECK ((event_type = ANY (ARRAY['ShiftOpened'::text, 'ShiftClosed'::text, 'OrderCreated'::text, 'OrderLineAdded'::text, 'OrderLineQuantityChanged'::text, 'OrderLineVoided'::text, 'PrecheckIssued'::text, 'PrecheckReprinted'::text, 'PrecheckCancelled'::text, 'CheckCreated'::text, 'CheckRefunded'::text, 'CheckReprinted'::text, 'PaymentCaptured'::text, 'PaymentRefunded'::text, 'CancellationRecorded'::text, 'RefundRecorded'::text, 'CheckClosed'::text, 'TicketIssued'::text, 'KitchenTicketStatusChanged'::text, 'ItemServed'::text, 'StockReceiptCaptured'::text, 'InventoryCountCaptured'::text, 'StockWriteOffCaptured'::text, 'ProductionCompleted'::text, 'StopListUpdated'::text, 'CatalogItemChangeSuggested'::text, 'RecipeChangeSuggested'::text, 'OrderClosed'::text, 'CashSessionOpened'::text, 'CashSessionClosed'::text, 'CashDrawerEventRecorded'::text, 'AuthSessionStarted'::text, 'AuthSessionRevoked'::text, 'DeviceRegistered'::text]))),
    CONSTRAINT cloud_edge_event_receipts_raw_payload_sha256_hex_check CHECK ((raw_payload_sha256_hex <> ''::text)),
    CONSTRAINT cloud_edge_event_receipts_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_edge_event_receipts_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT cloud_edge_event_receipts_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_edge_event_raw_payloads (
    receipt_id text NOT NULL,
    raw_payload jsonb NOT NULL,
    raw_payload_sha256_hex text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_edge_event_raw_payloads_raw_payload_sha256_hex_check CHECK ((raw_payload_sha256_hex <> ''::text)),
    CONSTRAINT cloud_edge_event_raw_payloads_pkey PRIMARY KEY (receipt_id),
    CONSTRAINT cloud_edge_event_raw_payloads_receipt_id_fkey FOREIGN KEY (receipt_id) REFERENCES public.cloud_edge_event_receipts(id) ON DELETE RESTRICT
);

CREATE TABLE public.cloud_goods (
    catalog_item_id text NOT NULL,
    restaurant_id text DEFAULT ''::text NOT NULL,
    stock_tracking_mode text DEFAULT 'none'::text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_goods_stock_tracking_mode_check CHECK ((stock_tracking_mode = ANY (ARRAY['none'::text, 'quantity'::text]))),
    CONSTRAINT cloud_goods_pkey PRIMARY KEY (catalog_item_id),
    CONSTRAINT cloud_goods_catalog_item_id_fkey FOREIGN KEY (catalog_item_id) REFERENCES public.cloud_catalog_items(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_master_data_delivery_states (
    node_device_id text NOT NULL,
    restaurant_id text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    effective_sha256 text DEFAULT ''::text NOT NULL,
    cloud_version bigint DEFAULT 0 NOT NULL,
    edge_ack_version bigint DEFAULT 0 NOT NULL,
    last_sync_at timestamp with time zone,
    last_error_code text DEFAULT ''::text NOT NULL,
    consecutive_failures integer DEFAULT 0 NOT NULL,
    next_retry_at timestamp with time zone,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_master_data_delivery_states_cloud_version_check CHECK ((cloud_version >= 0)),
    CONSTRAINT cloud_master_data_delivery_states_consecutive_failures_check CHECK ((consecutive_failures >= 0)),
    CONSTRAINT cloud_master_data_delivery_states_edge_ack_version_check CHECK ((edge_ack_version >= 0)),
    CONSTRAINT cloud_master_data_delivery_states_node_device_id_check CHECK ((node_device_id <> ''::text)),
    CONSTRAINT cloud_master_data_delivery_states_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_master_data_delivery_states_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'synced'::text, 'error'::text]))),
    CONSTRAINT cloud_master_data_delivery_states_pkey PRIMARY KEY (node_device_id)
);

CREATE TABLE public.cloud_master_data_packages (
    stream_name text NOT NULL,
    node_device_id text DEFAULT ''::text NOT NULL,
    restaurant_id text,
    sync_mode text NOT NULL,
    full_snapshot_reason text DEFAULT ''::text NOT NULL,
    cloud_version bigint NOT NULL,
    checkpoint_token text,
    cloud_updated_at timestamp with time zone,
    payload_json jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_master_data_packages_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_master_data_packages_full_snapshot_reason_check CHECK ((full_snapshot_reason = ANY (ARRAY[''::text, 'terminal_restaurant_changed'::text, 'node_role_changed'::text]))),
    CONSTRAINT cloud_master_data_packages_stream_name_check CHECK ((stream_name = ANY (ARRAY['restaurants'::text, 'devices'::text, 'staff'::text, 'floor'::text, 'catalog'::text, 'menu'::text, 'pricing_policy'::text, 'recipes'::text, 'inventory_reference'::text, 'currencies'::text, 'proposal_feedback'::text, 'receipt_templates'::text, 'printers'::text, 'sales_points'::text, 'restaurant_sections'::text]))),
    CONSTRAINT cloud_master_data_packages_sync_mode_check CHECK ((sync_mode = ANY (ARRAY['full_snapshot'::text, 'incremental'::text]))),
    CONSTRAINT cloud_master_data_packages_pkey PRIMARY KEY (stream_name, node_device_id)
);

CREATE TABLE public.cloud_master_data_publications (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    version bigint NOT NULL,
    status text NOT NULL,
    cloud_version bigint NOT NULL,
    published_at timestamp with time zone NOT NULL,
    published_by text NOT NULL,
    package_json jsonb NOT NULL,
    package_sha256 text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_master_data_publications_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_master_data_publications_package_sha256_check CHECK ((package_sha256 <> ''::text)),
    CONSTRAINT cloud_master_data_publications_published_by_check CHECK ((published_by <> ''::text)),
    CONSTRAINT cloud_master_data_publications_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_master_data_publications_status_check CHECK ((status = ANY (ARRAY['published'::text, 'archived'::text]))),
    CONSTRAINT cloud_master_data_publications_version_check CHECK ((version > 0)),
    CONSTRAINT cloud_master_data_publications_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_master_data_publications_restaurant_id_version_key UNIQUE (restaurant_id, version)
);

CREATE TABLE public.cloud_menu_items (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    catalog_item_id text NOT NULL,
    category_id text,
    tag_id text,
    tax_profile_id text,
    name text NOT NULL,
    price bigint NOT NULL,
    currency text NOT NULL,
    status text NOT NULL,
    runtime_status text DEFAULT 'available'::text NOT NULL,
    availability_json jsonb DEFAULT '{}'::jsonb NOT NULL,
    station_routing_key text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    CONSTRAINT cloud_menu_items_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_menu_items_currency_check CHECK ((currency ~ '^[A-Z]{3}$'::text)),
    CONSTRAINT cloud_menu_items_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_menu_items_price_check CHECK ((price >= 0)),
    CONSTRAINT cloud_menu_items_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_menu_items_runtime_status_check CHECK ((runtime_status = ANY (ARRAY['available'::text, 'unavailable'::text, 'hidden'::text]))),
    CONSTRAINT cloud_menu_items_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_menu_items_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_menu_items_catalog_item_id_fkey FOREIGN KEY (catalog_item_id) REFERENCES public.cloud_catalog_items(id),
    CONSTRAINT cloud_menu_items_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.cloud_categories(id),
    CONSTRAINT cloud_menu_items_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.cloud_catalog_tags(id)
);

CREATE TABLE public.cloud_menu_location_assignments (
    menu_item_id text NOT NULL,
    location_id text NOT NULL,
    active boolean DEFAULT true NOT NULL,
    CONSTRAINT cloud_menu_location_assignments_location_id_check CHECK ((location_id <> ''::text)),
    CONSTRAINT cloud_menu_location_assignments_pkey PRIMARY KEY (menu_item_id, location_id),
    CONSTRAINT cloud_menu_location_assignments_menu_item_id_fkey FOREIGN KEY (menu_item_id) REFERENCES public.cloud_menu_items(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_modifier_groups (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    required boolean DEFAULT false NOT NULL,
    min_count bigint DEFAULT 0 NOT NULL,
    max_count bigint DEFAULT 1 NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_modifier_groups_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_modifier_groups_max_count_check CHECK ((max_count >= 0)),
    CONSTRAINT cloud_modifier_groups_min_count_check CHECK ((min_count >= 0)),
    CONSTRAINT cloud_modifier_groups_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_modifier_groups_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_modifier_groups_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_modifier_groups_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_menu_item_modifier_groups (
    menu_item_id text NOT NULL,
    modifier_group_id text NOT NULL,
    sort_order bigint DEFAULT 0 NOT NULL,
    CONSTRAINT cloud_menu_item_modifier_groups_pkey PRIMARY KEY (menu_item_id, modifier_group_id),
    CONSTRAINT cloud_menu_item_modifier_groups_menu_item_id_fkey FOREIGN KEY (menu_item_id) REFERENCES public.cloud_menu_items(id) ON DELETE CASCADE,
    CONSTRAINT cloud_menu_item_modifier_groups_modifier_group_id_fkey FOREIGN KEY (modifier_group_id) REFERENCES public.cloud_modifier_groups(id) ON DELETE RESTRICT
);

CREATE TABLE public.cloud_modifier_group_bindings (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    modifier_group_id text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    sort_order bigint DEFAULT 0 NOT NULL,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_modifier_group_bindings_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_modifier_group_bindings_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_modifier_group_bindings_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_modifier_group_bindings_target_id_check CHECK ((target_id <> ''::text)),
    CONSTRAINT cloud_modifier_group_bindings_target_type_check CHECK ((target_type = ANY (ARRAY['menu_item'::text, 'catalog_item'::text, 'folder'::text, 'tag'::text]))),
    CONSTRAINT cloud_modifier_group_bindings_modifier_group_id_target_type_key UNIQUE (modifier_group_id, target_type, target_id),
    CONSTRAINT cloud_modifier_group_bindings_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_modifier_group_bindings_modifier_group_id_fkey FOREIGN KEY (modifier_group_id) REFERENCES public.cloud_modifier_groups(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_modifier_options (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    modifier_group_id text NOT NULL,
    linked_catalog_item_id text,
    name text NOT NULL,
    price_minor bigint DEFAULT 0 NOT NULL,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_modifier_options_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_modifier_options_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_modifier_options_price_minor_check CHECK ((price_minor >= 0)),
    CONSTRAINT cloud_modifier_options_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_modifier_options_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_modifier_options_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_modifier_options_linked_catalog_item_id_fkey FOREIGN KEY (linked_catalog_item_id) REFERENCES public.cloud_catalog_items(id),
    CONSTRAINT cloud_modifier_options_modifier_group_id_fkey FOREIGN KEY (modifier_group_id) REFERENCES public.cloud_modifier_groups(id)
);

CREATE TABLE public.cloud_operational_events (
    id text NOT NULL,
    receipt_id text NOT NULL,
    idempotency_key text NOT NULL,
    restaurant_id text NOT NULL,
    device_id text NOT NULL,
    command_id text NOT NULL,
    event_id text NOT NULL,
    edge_event_id text NOT NULL,
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    envelope_version text NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    cloud_received_at timestamp with time zone NOT NULL,
    raw_payload_sha256_hex text NOT NULL,
    replay_status text DEFAULT 'accepted'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cloud_operational_events_aggregate_id_check CHECK ((aggregate_id <> ''::text)),
    CONSTRAINT cloud_operational_events_aggregate_type_check CHECK ((aggregate_type <> ''::text)),
    CONSTRAINT cloud_operational_events_command_id_check CHECK ((command_id <> ''::text)),
    CONSTRAINT cloud_operational_events_device_id_check CHECK ((device_id <> ''::text)),
    CONSTRAINT cloud_operational_events_edge_event_id_check CHECK ((edge_event_id <> ''::text)),
    CONSTRAINT cloud_operational_events_envelope_version_check CHECK ((envelope_version = '1'::text)),
    CONSTRAINT cloud_operational_events_event_id_check CHECK ((event_id <> ''::text)),
    CONSTRAINT cloud_operational_events_event_type_check CHECK ((event_type <> ''::text)),
    CONSTRAINT cloud_operational_events_raw_payload_sha256_hex_check CHECK ((raw_payload_sha256_hex <> ''::text)),
    CONSTRAINT cloud_operational_events_replay_status_check CHECK ((replay_status = 'accepted'::text)),
    CONSTRAINT cloud_operational_events_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_operational_events_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT cloud_operational_events_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_operational_events_receipt_id_key UNIQUE (receipt_id),
    CONSTRAINT cloud_operational_events_receipt_id_fkey FOREIGN KEY (receipt_id) REFERENCES public.cloud_edge_event_receipts(id) ON DELETE RESTRICT
);

CREATE TABLE public.cloud_pricing_policies (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    name text NOT NULL,
    kind text NOT NULL,
    scope text NOT NULL,
    amount_kind text NOT NULL,
    amount_minor bigint DEFAULT 0 NOT NULL,
    value_basis_points bigint DEFAULT 0 NOT NULL,
    application_index bigint NOT NULL,
    manual boolean DEFAULT false NOT NULL,
    requires_permission text,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_pricing_policies_amount_kind_check CHECK ((amount_kind = ANY (ARRAY['percentage'::text, 'fixed'::text]))),
    CONSTRAINT cloud_pricing_policies_amount_minor_check CHECK ((amount_minor >= 0)),
    CONSTRAINT cloud_pricing_policies_application_index_check CHECK ((application_index > 0)),
    CONSTRAINT cloud_pricing_policies_check CHECK ((((amount_kind = 'fixed'::text) AND (amount_minor >= 0) AND (value_basis_points = 0)) OR ((amount_kind = 'percentage'::text) AND (value_basis_points > 0) AND (amount_minor = 0)))),
    CONSTRAINT cloud_pricing_policies_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_pricing_policies_kind_check CHECK ((kind = ANY (ARRAY['discount'::text, 'surcharge'::text]))),
    CONSTRAINT cloud_pricing_policies_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_pricing_policies_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_pricing_policies_scope_check CHECK ((scope = ANY (ARRAY['line'::text, 'order'::text]))),
    CONSTRAINT cloud_pricing_policies_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_pricing_policies_value_basis_points_check CHECK ((value_basis_points >= 0)),
    CONSTRAINT cloud_pricing_policies_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_printers (
    id text NOT NULL,
    org_id text DEFAULT ''::text NOT NULL,
    restaurant_id text NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    address text DEFAULT ''::text NOT NULL,
    port integer,
    document_types text DEFAULT '[]'::text NOT NULL,
    codepage text DEFAULT ''::text NOT NULL,
    paper_cut_type text DEFAULT 'partial'::text NOT NULL,
    cpl integer DEFAULT 42 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cloud_printers_codepage_check CHECK ((codepage = ANY (ARRAY[''::text, 'cp437'::text, 'cp866'::text]))),
    CONSTRAINT cloud_printers_cpl_check CHECK ((cpl = ANY (ARRAY[32, 42, 48, 56, 80]))),
    CONSTRAINT cloud_printers_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_printers_paper_cut_type_check CHECK ((paper_cut_type = ANY (ARRAY['full'::text, 'partial'::text]))),
    CONSTRAINT cloud_printers_type_check CHECK ((type = ANY (ARRAY['tcp'::text, 'usb'::text]))),
    CONSTRAINT cloud_printers_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_projection_event_type_stats (
    restaurant_id text NOT NULL,
    device_id text NOT NULL,
    event_type text NOT NULL,
    event_count bigint NOT NULL,
    first_occurred_at timestamp with time zone NOT NULL,
    last_occurred_at timestamp with time zone NOT NULL,
    last_cloud_received_at timestamp with time zone NOT NULL,
    last_event_id text NOT NULL,
    last_command_id text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_projection_event_type_stats_device_id_check CHECK ((device_id <> ''::text)),
    CONSTRAINT cloud_projection_event_type_stats_event_count_check CHECK ((event_count >= 0)),
    CONSTRAINT cloud_projection_event_type_stats_event_type_check CHECK ((event_type <> ''::text)),
    CONSTRAINT cloud_projection_event_type_stats_last_command_id_check CHECK ((last_command_id <> ''::text)),
    CONSTRAINT cloud_projection_event_type_stats_last_event_id_check CHECK ((last_event_id <> ''::text)),
    CONSTRAINT cloud_projection_event_type_stats_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_projection_event_type_stats_pkey PRIMARY KEY (restaurant_id, device_id, event_type)
);

CREATE TABLE public.cloud_projection_financial_operations (
    operation_id text NOT NULL,
    edge_operation_id text NOT NULL,
    event_id text NOT NULL,
    receipt_id text NOT NULL,
    restaurant_id text NOT NULL,
    device_id text NOT NULL,
    node_device_id text,
    client_device_id text,
    actor_employee_id text,
    session_id text,
    shift_id text NOT NULL,
    original_shift_id text NOT NULL,
    check_id text NOT NULL,
    precheck_id text NOT NULL,
    operation_type text NOT NULL,
    operation_kind text NOT NULL,
    amount bigint NOT NULL,
    currency text NOT NULL,
    business_date_local text NOT NULL,
    inventory_disposition text NOT NULL,
    reason text NOT NULL,
    created_by_employee_id text,
    approved_by_employee_id text,
    snapshot_json jsonb NOT NULL,
    operation_created_at timestamp with time zone NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    cloud_received_at timestamp with time zone NOT NULL,
    raw_payload_sha256_hex text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_projection_financial_operat_approved_by_employee_id_check CHECK (((approved_by_employee_id IS NULL) OR (approved_by_employee_id <> ''::text))),
    CONSTRAINT cloud_projection_financial_operati_created_by_employee_id_check CHECK (((created_by_employee_id IS NULL) OR (created_by_employee_id <> ''::text))),
    CONSTRAINT cloud_projection_financial_operati_raw_payload_sha256_hex_check CHECK ((raw_payload_sha256_hex <> ''::text)),
    CONSTRAINT cloud_projection_financial_operatio_inventory_disposition_check CHECK ((inventory_disposition = ANY (ARRAY['no_stock_effect'::text, 'return_to_stock'::text, 'write_off_waste'::text, 'manual_review'::text]))),
    CONSTRAINT cloud_projection_financial_operations_actor_employee_id_check CHECK (((actor_employee_id IS NULL) OR (actor_employee_id <> ''::text))),
    CONSTRAINT cloud_projection_financial_operations_amount_check CHECK ((amount > 0)),
    CONSTRAINT cloud_projection_financial_operations_business_date_local_check CHECK ((business_date_local ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'::text)),
    CONSTRAINT cloud_projection_financial_operations_check_id_check CHECK ((check_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_client_device_id_check CHECK (((client_device_id IS NULL) OR (client_device_id <> ''::text))),
    CONSTRAINT cloud_projection_financial_operations_currency_check CHECK ((currency ~ '^[A-Z]{3}$'::text)),
    CONSTRAINT cloud_projection_financial_operations_device_id_check CHECK ((device_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_edge_operation_id_check CHECK ((edge_operation_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_event_id_check CHECK ((event_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_node_device_id_check CHECK (((node_device_id IS NULL) OR (node_device_id <> ''::text))),
    CONSTRAINT cloud_projection_financial_operations_operation_id_check CHECK ((operation_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_operation_kind_check CHECK ((operation_kind = ANY (ARRAY['full'::text, 'partial'::text]))),
    CONSTRAINT cloud_projection_financial_operations_operation_type_check CHECK ((operation_type = ANY (ARRAY['cancellation'::text, 'refund'::text]))),
    CONSTRAINT cloud_projection_financial_operations_original_shift_id_check CHECK ((original_shift_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_precheck_id_check CHECK ((precheck_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_reason_check CHECK ((reason <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_session_id_check CHECK (((session_id IS NULL) OR (session_id <> ''::text))),
    CONSTRAINT cloud_projection_financial_operations_shift_id_check CHECK ((shift_id <> ''::text)),
    CONSTRAINT cloud_projection_financial_operations_event_id_key UNIQUE (event_id),
    CONSTRAINT cloud_projection_financial_operations_pkey PRIMARY KEY (operation_id),
    CONSTRAINT cloud_projection_financial_operations_receipt_id_key UNIQUE (receipt_id),
    CONSTRAINT cloud_projection_financial_operations_receipt_id_fkey FOREIGN KEY (receipt_id) REFERENCES public.cloud_edge_event_receipts(id) ON DELETE RESTRICT
);

CREATE TABLE public.cloud_projection_shift_finance (
    restaurant_id text NOT NULL,
    device_id text NOT NULL,
    shift_id text NOT NULL,
    payments_captured_count bigint DEFAULT 0 NOT NULL,
    payments_captured_total bigint DEFAULT 0 NOT NULL,
    checks_created_count bigint DEFAULT 0 NOT NULL,
    checks_total_amount bigint DEFAULT 0 NOT NULL,
    last_event_id text NOT NULL,
    last_command_id text NOT NULL,
    last_occurred_at timestamp with time zone NOT NULL,
    last_cloud_received_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    payments_refunded_count bigint DEFAULT 0 NOT NULL,
    payments_refunded_total bigint DEFAULT 0 NOT NULL,
    checks_refunded_count bigint DEFAULT 0 NOT NULL,
    checks_refunded_total bigint DEFAULT 0 NOT NULL,
    CONSTRAINT cloud_projection_shift_finance_checks_created_count_check CHECK ((checks_created_count >= 0)),
    CONSTRAINT cloud_projection_shift_finance_checks_refunded_count_check CHECK ((checks_refunded_count >= 0)),
    CONSTRAINT cloud_projection_shift_finance_device_id_check CHECK ((device_id <> ''::text)),
    CONSTRAINT cloud_projection_shift_finance_last_command_id_check CHECK ((last_command_id <> ''::text)),
    CONSTRAINT cloud_projection_shift_finance_last_event_id_check CHECK ((last_event_id <> ''::text)),
    CONSTRAINT cloud_projection_shift_finance_payments_captured_count_check CHECK ((payments_captured_count >= 0)),
    CONSTRAINT cloud_projection_shift_finance_payments_refunded_count_check CHECK ((payments_refunded_count >= 0)),
    CONSTRAINT cloud_projection_shift_finance_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_projection_shift_finance_shift_id_check CHECK ((shift_id <> ''::text)),
    CONSTRAINT cloud_projection_shift_finance_pkey PRIMARY KEY (restaurant_id, device_id, shift_id)
);

CREATE TABLE public.cloud_projection_stop_list_updates (
    source_event_id text NOT NULL,
    queue_id text NOT NULL,
    restaurant_id text NOT NULL,
    device_id text NOT NULL,
    stop_list_id text NOT NULL,
    warehouse_id text,
    catalog_item_id text NOT NULL,
    available_quantity numeric(14,3),
    active boolean NOT NULL,
    conflict_policy text NOT NULL,
    source text NOT NULL,
    reason text,
    projection_action text NOT NULL,
    review_status text DEFAULT 'pending'::text NOT NULL,
    review_comment text DEFAULT ''::text NOT NULL,
    reviewed_by_employee_id text DEFAULT ''::text NOT NULL,
    reviewed_at timestamp with time zone,
    assigned_to_employee_id text DEFAULT ''::text NOT NULL,
    assigned_by_employee_id text DEFAULT ''::text NOT NULL,
    assigned_at timestamp with time zone,
    assignment_note text DEFAULT ''::text NOT NULL,
    applied_stop_list_id text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    projected_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cloud_projection_stop_list_updates_catalog_item_id_check CHECK ((catalog_item_id <> ''::text)),
    CONSTRAINT cloud_projection_stop_list_updates_conflict_policy_check CHECK ((conflict_policy = ANY (ARRAY['cloud_wins'::text, 'edge_overlay_until_next_publication'::text, 'edge_overlay_requires_manager_review'::text]))),
    CONSTRAINT cloud_projection_stop_list_updates_device_id_check CHECK ((device_id <> ''::text)),
    CONSTRAINT cloud_projection_stop_list_updates_projection_action_check CHECK ((projection_action = ANY (ARRAY['applied_edge_overlay'::text, 'ignored_cloud_wins'::text, 'requires_manager_review'::text]))),
    CONSTRAINT cloud_projection_stop_list_updates_queue_id_check CHECK ((queue_id <> ''::text)),
    CONSTRAINT cloud_projection_stop_list_updates_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_projection_stop_list_updates_review_status_check CHECK ((review_status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'changes_requested'::text]))),
    CONSTRAINT cloud_projection_stop_list_updates_source_check CHECK ((source <> ''::text)),
    CONSTRAINT cloud_projection_stop_list_updates_source_event_id_check CHECK ((source_event_id <> ''::text)),
    CONSTRAINT cloud_projection_stop_list_updates_stop_list_id_check CHECK ((stop_list_id <> ''::text)),
    CONSTRAINT cloud_projection_stop_list_updates_pkey PRIMARY KEY (source_event_id)
);

CREATE TABLE public.cloud_receipt_templates (
    id text NOT NULL,
    org_id text DEFAULT ''::text NOT NULL,
    restaurant_id text,
    document_type text NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    content text NOT NULL,
    level integer DEFAULT 1 NOT NULL,
    cpl integer NOT NULL,
    printer_class text DEFAULT 'generic'::text NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cloud_receipt_templates_content_check CHECK ((content <> ''::text)),
    CONSTRAINT cloud_receipt_templates_cpl_check CHECK ((cpl = ANY (ARRAY[32, 40, 48, 58]))),
    CONSTRAINT cloud_receipt_templates_document_type_check CHECK ((document_type = ANY (ARRAY['precheck'::text, 'check_nonfiscal'::text, 'ticket'::text, 'kitchen_service'::text, 'cash_in_out'::text, 'acceptance'::text]))),
    CONSTRAINT cloud_receipt_templates_level_check CHECK ((level = ANY (ARRAY[1, 2]))),
    CONSTRAINT cloud_receipt_templates_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_receipt_templates_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_recipe_items (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    recipe_owner_catalog_item_id text NOT NULL,
    component_catalog_item_id text NOT NULL,
    quantity bigint NOT NULL,
    unit text NOT NULL,
    loss_percent bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_recipe_items_loss_percent_check CHECK (((loss_percent >= 0) AND (loss_percent <= 100))),
    CONSTRAINT cloud_recipe_items_quantity_check CHECK ((quantity > 0)),
    CONSTRAINT cloud_recipe_items_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_recipe_items_unit_check CHECK ((unit <> ''::text)),
    CONSTRAINT cloud_recipe_items_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_recipe_items_recipe_owner_catalog_item_id_component_c_key UNIQUE (recipe_owner_catalog_item_id, component_catalog_item_id),
    CONSTRAINT cloud_recipe_items_component_catalog_item_id_fkey FOREIGN KEY (component_catalog_item_id) REFERENCES public.cloud_catalog_items(id),
    CONSTRAINT cloud_recipe_items_recipe_owner_catalog_item_id_fkey FOREIGN KEY (recipe_owner_catalog_item_id) REFERENCES public.cloud_catalog_items(id)
);

CREATE TABLE public.cloud_recipe_suggestions (
    id text NOT NULL,
    suggestion_id text NOT NULL,
    restaurant_id text NOT NULL,
    recipe_version_id text,
    owner_catalog_item_id text,
    owner_catalog_suggestion_id text,
    proposal_group_id text,
    action text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    prep_time_delta_minutes bigint DEFAULT 0 NOT NULL,
    status text NOT NULL,
    review_comment text DEFAULT ''::text NOT NULL,
    reviewed_by_employee_id text DEFAULT ''::text NOT NULL,
    reviewed_at timestamp with time zone,
    assigned_to_employee_id text DEFAULT ''::text NOT NULL,
    assigned_by_employee_id text DEFAULT ''::text NOT NULL,
    assigned_at timestamp with time zone,
    assignment_note text DEFAULT ''::text NOT NULL,
    source_event_id text DEFAULT ''::text NOT NULL,
    suggested_at timestamp with time zone NOT NULL,
    cloud_received_at timestamp with time zone NOT NULL,
    payload_json jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_recipe_suggestions_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'changes_requested'::text]))),
    CONSTRAINT cloud_recipe_suggestions_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_recipe_suggestions_suggestion_id_key UNIQUE (suggestion_id)
);

CREATE TABLE public.cloud_recipe_suggestion_changes (
    id text NOT NULL,
    recipe_suggestion_id text NOT NULL,
    line_id text DEFAULT ''::text NOT NULL,
    action text NOT NULL,
    from_catalog_item_id text DEFAULT ''::text NOT NULL,
    to_catalog_item_id text DEFAULT ''::text NOT NULL,
    quantity text DEFAULT ''::text NOT NULL,
    unit_code text DEFAULT ''::text NOT NULL,
    loss_percent text DEFAULT ''::text NOT NULL,
    sort_order bigint DEFAULT 0 NOT NULL,
    payload_json jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_recipe_suggestion_changes_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_recipe_suggestion_changes_recipe_suggestion_id_fkey FOREIGN KEY (recipe_suggestion_id) REFERENCES public.cloud_recipe_suggestions(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_recipe_versions (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    owner_catalog_item_id text NOT NULL,
    version bigint NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    yield_quantity bigint DEFAULT 1 NOT NULL,
    yield_unit text NOT NULL,
    created_by_employee_id text,
    submitted_by_employee_id text,
    approved_by_employee_id text,
    submitted_at timestamp with time zone,
    approved_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_recipe_versions_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_recipe_versions_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_recipe_versions_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'review_pending'::text, 'active'::text, 'archived'::text]))),
    CONSTRAINT cloud_recipe_versions_version_check CHECK ((version > 0)),
    CONSTRAINT cloud_recipe_versions_yield_quantity_check CHECK ((yield_quantity > 0)),
    CONSTRAINT cloud_recipe_versions_yield_unit_check CHECK ((yield_unit <> ''::text)),
    CONSTRAINT cloud_recipe_versions_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_recipe_versions_restaurant_id_owner_catalog_item_id_v_key UNIQUE (restaurant_id, owner_catalog_item_id, version),
    CONSTRAINT cloud_recipe_versions_owner_catalog_item_id_fkey FOREIGN KEY (owner_catalog_item_id) REFERENCES public.cloud_catalog_items(id)
);

CREATE TABLE public.cloud_recipe_lines (
    id text NOT NULL,
    recipe_version_id text NOT NULL,
    component_catalog_item_id text NOT NULL,
    quantity bigint NOT NULL,
    unit text NOT NULL,
    loss_percent bigint DEFAULT 0 NOT NULL,
    sort_order bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_recipe_lines_loss_percent_check CHECK (((loss_percent >= 0) AND (loss_percent <= 100))),
    CONSTRAINT cloud_recipe_lines_quantity_check CHECK ((quantity > 0)),
    CONSTRAINT cloud_recipe_lines_unit_check CHECK ((unit <> ''::text)),
    CONSTRAINT cloud_recipe_lines_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_recipe_lines_recipe_version_id_component_catalog_item_key UNIQUE (recipe_version_id, component_catalog_item_id),
    CONSTRAINT cloud_recipe_lines_component_catalog_item_id_fkey FOREIGN KEY (component_catalog_item_id) REFERENCES public.cloud_catalog_items(id),
    CONSTRAINT cloud_recipe_lines_recipe_version_id_fkey FOREIGN KEY (recipe_version_id) REFERENCES public.cloud_recipe_versions(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_restaurants (
    id text NOT NULL,
    name text NOT NULL,
    timezone text NOT NULL,
    currency text NOT NULL,
    business_day_mode text NOT NULL,
    business_day_boundary_local_time text NOT NULL,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_restaurants_business_day_boundary_local_time_check CHECK ((business_day_boundary_local_time ~ '^[0-2][0-9]:[0-5][0-9]'::text)),
    CONSTRAINT cloud_restaurants_business_day_mode_check CHECK ((business_day_mode = ANY (ARRAY['standard'::text, '24_7'::text]))),
    CONSTRAINT cloud_restaurants_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_restaurants_currency_check CHECK ((currency ~ '^[A-Z]{3}$'::text)),
    CONSTRAINT cloud_restaurants_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_restaurants_status_check CHECK ((status = ANY (ARRAY['active'::text, 'archived'::text]))),
    CONSTRAINT cloud_restaurants_timezone_check CHECK ((timezone <> ''::text)),
    CONSTRAINT cloud_restaurants_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_edge_nodes (
    id text NOT NULL,
    restaurant_id text,
    node_device_id text NOT NULL,
    display_name text NOT NULL,
    status text NOT NULL,
    credentials_hash text,
    last_seen_at timestamp with time zone,
    assigned_at timestamp with time zone,
    revoked_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_edge_nodes_credentials_hash_check CHECK (((credentials_hash IS NULL) OR (credentials_hash <> ''::text))),
    CONSTRAINT cloud_edge_nodes_display_name_check CHECK ((display_name <> ''::text)),
    CONSTRAINT cloud_edge_nodes_node_device_id_check CHECK ((node_device_id <> ''::text)),
    CONSTRAINT cloud_edge_nodes_status_check CHECK ((status = ANY (ARRAY['unassigned'::text, 'assigned'::text, 'revoked'::text]))),
    CONSTRAINT cloud_edge_nodes_node_device_id_key UNIQUE (node_device_id),
    CONSTRAINT cloud_edge_nodes_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_edge_nodes_restaurant_id_fkey FOREIGN KEY (restaurant_id) REFERENCES public.cloud_restaurants(id)
);

CREATE TABLE public.cloud_halls (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_halls_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_halls_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_halls_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_halls_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_halls_restaurant_id_fkey FOREIGN KEY (restaurant_id) REFERENCES public.cloud_restaurants(id)
);

CREATE TABLE public.cloud_restaurant_sections (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    name text NOT NULL,
    mode text NOT NULL,
    hall_id text,
    kitchen_routing_key text,
    warehouse_id text,
    is_default boolean DEFAULT false NOT NULL,
    is_active boolean DEFAULT false NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_restaurant_sections_mode_check CHECK ((mode = ANY (ARRAY['hall_section'::text, 'kitchen_workshop'::text]))),
    CONSTRAINT cloud_restaurant_sections_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_restaurant_sections_version_check CHECK ((version > 0)),
    CONSTRAINT cloud_restaurant_sections_hall_mode_check CHECK (((mode = 'hall_section'::text AND kitchen_routing_key IS NULL) OR (mode = 'kitchen_workshop'::text AND hall_id IS NULL))),
    CONSTRAINT cloud_restaurant_sections_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_restaurant_sections_hall_id_fkey FOREIGN KEY (hall_id) REFERENCES public.cloud_halls(id),
    CONSTRAINT cloud_restaurant_sections_restaurant_id_fkey FOREIGN KEY (restaurant_id) REFERENCES public.cloud_restaurants(id)
);

CREATE TABLE public.cloud_pairing_codes (
    id text NOT NULL,
    pairing_code_hash text NOT NULL,
    pairing_key text NOT NULL,
    restaurant_id text NOT NULL,
    node_device_id text,
    cloud_url text NOT NULL,
    status text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_pairing_codes_cloud_url_check CHECK ((cloud_url <> ''::text)),
    CONSTRAINT cloud_pairing_codes_pairing_code_hash_check CHECK ((pairing_code_hash <> ''::text)),
    CONSTRAINT cloud_pairing_codes_pairing_key_check CHECK ((pairing_key <> ''::text)),
    CONSTRAINT cloud_pairing_codes_status_check CHECK ((status = ANY (ARRAY['active'::text, 'consumed'::text, 'expired'::text, 'revoked'::text]))),
    CONSTRAINT cloud_pairing_codes_pairing_code_hash_key UNIQUE (pairing_code_hash),
    CONSTRAINT cloud_pairing_codes_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_pairing_codes_restaurant_id_fkey FOREIGN KEY (restaurant_id) REFERENCES public.cloud_restaurants(id)
);

CREATE TABLE public.cloud_review_assignment_audit_events (
    event_id text NOT NULL,
    command_id text NOT NULL,
    review_type text NOT NULL,
    review_id text NOT NULL,
    action text NOT NULL,
    actor_employee_id text NOT NULL,
    target_employee_id text DEFAULT ''::text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_review_assignment_audit_events_action_check CHECK ((action = ANY (ARRAY['assigned'::text, 'unassigned'::text]))),
    CONSTRAINT cloud_review_assignment_audit_events_actor_employee_id_check CHECK ((actor_employee_id <> ''::text)),
    CONSTRAINT cloud_review_assignment_audit_events_command_id_check CHECK ((command_id <> ''::text)),
    CONSTRAINT cloud_review_assignment_audit_events_event_id_check CHECK ((event_id <> ''::text)),
    CONSTRAINT cloud_review_assignment_audit_events_review_id_check CHECK ((review_id <> ''::text)),
    CONSTRAINT cloud_review_assignment_audit_events_review_type_check CHECK ((review_type = ANY (ARRAY['stop_list_update'::text, 'catalog_suggestion'::text, 'recipe_suggestion'::text]))),
    CONSTRAINT cloud_review_assignment_audit_events_command_id_key UNIQUE (command_id),
    CONSTRAINT cloud_review_assignment_audit_events_pkey PRIMARY KEY (event_id)
);

CREATE TABLE public.cloud_roles (
    id text NOT NULL,
    name text NOT NULL,
    permissions_json jsonb NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    CONSTRAINT cloud_roles_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_roles_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_roles_name_key UNIQUE (name),
    CONSTRAINT cloud_roles_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_employees (
    id text NOT NULL,
    role_id text NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    pin_hash text NOT NULL,
    pin_credential_version bigint DEFAULT 1 NOT NULL,
    permission_snapshot_json jsonb NOT NULL,
    suspended_at timestamp with time zone,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    CONSTRAINT cloud_employees_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_employees_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_employees_pin_credential_version_check CHECK ((pin_credential_version > 0)),
    CONSTRAINT cloud_employees_pin_hash_check CHECK ((pin_hash <> ''::text)),
    CONSTRAINT cloud_employees_status_check CHECK ((status = ANY (ARRAY['active'::text, 'suspended'::text, 'archived'::text]))),
    CONSTRAINT cloud_employees_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_employees_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.cloud_roles(id)
);

CREATE TABLE public.cloud_employee_restaurant_memberships (
    employee_id text NOT NULL,
    restaurant_id text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_employee_restaurant_memberships_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT cloud_employee_restaurant_memberships_pkey PRIMARY KEY (employee_id, restaurant_id),
    CONSTRAINT cloud_employee_restaurant_memberships_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.cloud_employees(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_semi_finished_products (
    catalog_item_id text NOT NULL,
    restaurant_id text DEFAULT ''::text NOT NULL,
    production_unit text DEFAULT 'portion'::text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_semi_finished_products_pkey PRIMARY KEY (catalog_item_id),
    CONSTRAINT cloud_semi_finished_products_catalog_item_id_fkey FOREIGN KEY (catalog_item_id) REFERENCES public.cloud_catalog_items(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_services (
    catalog_item_id text NOT NULL,
    restaurant_id text DEFAULT ''::text NOT NULL,
    fixed_unit text DEFAULT 'service'::text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_services_pkey PRIMARY KEY (catalog_item_id),
    CONSTRAINT cloud_services_catalog_item_id_fkey FOREIGN KEY (catalog_item_id) REFERENCES public.cloud_catalog_items(id) ON DELETE CASCADE
);

CREATE TABLE public.cloud_suggestion_review_events (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    suggestion_kind text NOT NULL,
    suggestion_id text NOT NULL,
    status text NOT NULL,
    reviewed_by_employee_id text DEFAULT ''::text NOT NULL,
    review_comment text DEFAULT ''::text NOT NULL,
    reviewed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_suggestion_review_events_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'changes_requested'::text]))),
    CONSTRAINT cloud_suggestion_review_events_suggestion_kind_check CHECK ((suggestion_kind = ANY (ARRAY['catalog'::text, 'recipe'::text]))),
    CONSTRAINT cloud_suggestion_review_events_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_sync_problem_events (
    id text NOT NULL,
    direction text NOT NULL,
    node_device_id text,
    restaurant_id text,
    client_item_id text,
    error_code text NOT NULL,
    error_message text NOT NULL,
    raw_payload text NOT NULL,
    raw_payload_sha256_hex text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_sync_problem_events_direction_check CHECK ((direction = ANY (ARRAY['edge_to_cloud'::text, 'cloud_to_edge'::text]))),
    CONSTRAINT cloud_sync_problem_events_error_code_check CHECK ((error_code <> ''::text)),
    CONSTRAINT cloud_sync_problem_events_error_message_check CHECK ((error_message <> ''::text)),
    CONSTRAINT cloud_sync_problem_events_raw_payload_sha256_hex_check CHECK ((raw_payload_sha256_hex <> ''::text)),
    CONSTRAINT cloud_sync_problem_events_pkey PRIMARY KEY (id)
);

CREATE TABLE public.cloud_tables (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    hall_id text,
    section_id text NOT NULL,
    name text NOT NULL,
    seats bigint DEFAULT 0 NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    status text NOT NULL,
    cloud_version bigint DEFAULT 1 NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_tables_cloud_version_check CHECK ((cloud_version > 0)),
    CONSTRAINT cloud_tables_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_tables_seats_check CHECK ((seats >= 0)),
    CONSTRAINT cloud_tables_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT cloud_tables_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_tables_hall_id_fkey FOREIGN KEY (hall_id) REFERENCES public.cloud_halls(id),
    CONSTRAINT cloud_tables_section_id_fkey FOREIGN KEY (section_id) REFERENCES public.cloud_restaurant_sections(id),
    CONSTRAINT cloud_tables_restaurant_id_fkey FOREIGN KEY (restaurant_id) REFERENCES public.cloud_restaurants(id)
);

CREATE TABLE public.cloud_sales_points (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    name text NOT NULL,
    analytics_tag text NOT NULL,
    default_table_id text NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_sales_points_analytics_tag_check CHECK ((analytics_tag <> ''::text)),
    CONSTRAINT cloud_sales_points_name_check CHECK ((name <> ''::text)),
    CONSTRAINT cloud_sales_points_version_check CHECK ((version > 0)),
    CONSTRAINT cloud_sales_points_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_sales_points_default_table_id_fkey FOREIGN KEY (default_table_id) REFERENCES public.cloud_tables(id),
    CONSTRAINT cloud_sales_points_restaurant_id_fkey FOREIGN KEY (restaurant_id) REFERENCES public.cloud_restaurants(id),
    CONSTRAINT cloud_sales_points_restaurant_analytics_tag_key UNIQUE (restaurant_id, analytics_tag)
);

CREATE TABLE public.cloud_unassigned_edge_nodes (
    id text NOT NULL,
    node_device_id text NOT NULL,
    claimed_cloud_url text NOT NULL,
    display_name text NOT NULL,
    app_version text DEFAULT ''::text NOT NULL,
    status text NOT NULL,
    first_seen_at timestamp with time zone NOT NULL,
    last_seen_at timestamp with time zone NOT NULL,
    assigned_restaurant_id text,
    assigned_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT cloud_unassigned_edge_nodes_claimed_cloud_url_check CHECK ((claimed_cloud_url <> ''::text)),
    CONSTRAINT cloud_unassigned_edge_nodes_display_name_check CHECK ((display_name <> ''::text)),
    CONSTRAINT cloud_unassigned_edge_nodes_node_device_id_check CHECK ((node_device_id <> ''::text)),
    CONSTRAINT cloud_unassigned_edge_nodes_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'assigned'::text, 'rejected'::text, 'expired'::text]))),
    CONSTRAINT cloud_unassigned_edge_nodes_node_device_id_key UNIQUE (node_device_id),
    CONSTRAINT cloud_unassigned_edge_nodes_pkey PRIMARY KEY (id),
    CONSTRAINT cloud_unassigned_edge_nodes_assigned_restaurant_id_fkey FOREIGN KEY (assigned_restaurant_id) REFERENCES public.cloud_restaurants(id)
);

CREATE TABLE public.inbox_events (
    id text NOT NULL,
    receipt_id text NOT NULL,
    idempotency_key text NOT NULL,
    tenant_id text NOT NULL,
    restaurant_id text NOT NULL,
    device_id text NOT NULL,
    employee_id text DEFAULT ''::text NOT NULL,
    command_id text NOT NULL,
    event_id text NOT NULL,
    edge_event_id text NOT NULL,
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    envelope_version text NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    cloud_received_at timestamp with time zone NOT NULL,
    raw_payload jsonb NOT NULL,
    raw_payload_sha256_hex text NOT NULL,
    processed_for_olap boolean DEFAULT false NOT NULL,
    olap_export_status text DEFAULT 'pending'::text NOT NULL,
    olap_export_attempts bigint DEFAULT 0 NOT NULL,
    olap_next_retry_at timestamp with time zone,
    olap_locked_at timestamp with time zone,
    olap_locked_by text,
    olap_processed_at timestamp with time zone,
    olap_last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT inbox_events_aggregate_id_check CHECK ((aggregate_id <> ''::text)),
    CONSTRAINT inbox_events_aggregate_type_check CHECK ((aggregate_type <> ''::text)),
    CONSTRAINT inbox_events_command_id_check CHECK ((command_id <> ''::text)),
    CONSTRAINT inbox_events_device_id_check CHECK ((device_id <> ''::text)),
    CONSTRAINT inbox_events_edge_event_id_check CHECK ((edge_event_id <> ''::text)),
    CONSTRAINT inbox_events_envelope_version_check CHECK ((envelope_version = '1'::text)),
    CONSTRAINT inbox_events_event_id_check CHECK ((event_id <> ''::text)),
    CONSTRAINT inbox_events_event_type_check CHECK ((event_type <> ''::text)),
    CONSTRAINT inbox_events_olap_export_attempts_check CHECK ((olap_export_attempts >= 0)),
    CONSTRAINT inbox_events_olap_export_status_check CHECK ((olap_export_status = ANY (ARRAY['pending'::text, 'processing'::text, 'processed'::text, 'failed'::text]))),
    CONSTRAINT inbox_events_raw_payload_sha256_hex_check CHECK ((raw_payload_sha256_hex <> ''::text)),
    CONSTRAINT inbox_events_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT inbox_events_tenant_id_check CHECK ((tenant_id <> ''::text)),
    CONSTRAINT inbox_events_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT inbox_events_pkey PRIMARY KEY (id),
    CONSTRAINT inbox_events_receipt_id_key UNIQUE (receipt_id),
    CONSTRAINT inbox_events_id_fkey FOREIGN KEY (id) REFERENCES public.cloud_edge_event_receipts(id) ON DELETE RESTRICT,
    CONSTRAINT inbox_events_receipt_id_fkey FOREIGN KEY (receipt_id) REFERENCES public.cloud_edge_event_receipts(id) ON DELETE RESTRICT
);

CREATE TABLE public.inventory_event_queue (
    id text NOT NULL,
    receipt_id text NOT NULL,
    restaurant_id text NOT NULL,
    warehouse_id text,
    device_id text NOT NULL,
    event_id text NOT NULL,
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    status text NOT NULL,
    attempts bigint DEFAULT 0 NOT NULL,
    next_retry_at timestamp with time zone,
    locked_at timestamp with time zone,
    locked_by text,
    processed_at timestamp with time zone,
    last_error text,
    occurred_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT inventory_event_queue_aggregate_id_check CHECK ((aggregate_id <> ''::text)),
    CONSTRAINT inventory_event_queue_aggregate_type_check CHECK ((aggregate_type <> ''::text)),
    CONSTRAINT inventory_event_queue_attempts_check CHECK ((attempts >= 0)),
    CONSTRAINT inventory_event_queue_device_id_check CHECK ((device_id <> ''::text)),
    CONSTRAINT inventory_event_queue_event_id_check CHECK ((event_id <> ''::text)),
    CONSTRAINT inventory_event_queue_event_type_check CHECK ((event_type <> ''::text)),
    CONSTRAINT inventory_event_queue_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT inventory_event_queue_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'processing'::text, 'processed'::text, 'failed'::text]))),
    CONSTRAINT inventory_event_queue_pkey PRIMARY KEY (id),
    CONSTRAINT inventory_event_queue_receipt_id_key UNIQUE (receipt_id),
    CONSTRAINT inventory_event_queue_receipt_id_fkey FOREIGN KEY (receipt_id) REFERENCES public.cloud_edge_event_receipts(id) ON DELETE RESTRICT
);

CREATE TABLE public.olap_backfill_jobs (
    id text NOT NULL,
    command_id text NOT NULL,
    stream text NOT NULL,
    status text NOT NULL,
    requested_from timestamp with time zone,
    requested_to timestamp with time zone,
    checkpoint_cursor text DEFAULT ''::text NOT NULL,
    batch_size integer DEFAULT 1000 NOT NULL,
    total_rows bigint DEFAULT 0 NOT NULL,
    processed_rows bigint DEFAULT 0 NOT NULL,
    last_error text DEFAULT ''::text NOT NULL,
    cancel_requested boolean DEFAULT false NOT NULL,
    reason text NOT NULL,
    requested_by text DEFAULT ''::text NOT NULL,
    locked_by text DEFAULT ''::text NOT NULL,
    locked_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT olap_backfill_jobs_batch_size_check CHECK ((batch_size > 0)),
    CONSTRAINT olap_backfill_jobs_command_id_check CHECK ((command_id <> ''::text)),
    CONSTRAINT olap_backfill_jobs_id_check CHECK ((id <> ''::text)),
    CONSTRAINT olap_backfill_jobs_processed_rows_check CHECK ((processed_rows >= 0)),
    CONSTRAINT olap_backfill_jobs_reason_check CHECK ((reason <> ''::text)),
    CONSTRAINT olap_backfill_jobs_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'running'::text, 'completed'::text, 'failed'::text, 'cancelled'::text]))),
    CONSTRAINT olap_backfill_jobs_stream_check CHECK ((stream = ANY (ARRAY['raw_business_events'::text, 'stock_moves'::text]))),
    CONSTRAINT olap_backfill_jobs_total_rows_check CHECK ((total_rows >= 0)),
    CONSTRAINT olap_backfill_jobs_command_id_key UNIQUE (command_id),
    CONSTRAINT olap_backfill_jobs_pkey PRIMARY KEY (id)
);

CREATE TABLE public.olap_export_checkpoints (
    id text NOT NULL,
    worker_id text DEFAULT ''::text NOT NULL,
    last_exported_inbox_id text DEFAULT ''::text NOT NULL,
    last_exported_event_id text DEFAULT ''::text NOT NULL,
    last_exported_at timestamp with time zone,
    last_error text DEFAULT ''::text NOT NULL,
    consecutive_failures bigint DEFAULT 0 NOT NULL,
    next_retry_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT olap_export_checkpoints_consecutive_failures_check CHECK ((consecutive_failures >= 0)),
    CONSTRAINT olap_export_checkpoints_pkey PRIMARY KEY (id)
);

CREATE TABLE public.olap_export_retry_commands (
    command_id text NOT NULL,
    stream text NOT NULL,
    mode text NOT NULL,
    reason text NOT NULL,
    accepted boolean DEFAULT true NOT NULL,
    checkpoint_before text DEFAULT ''::text NOT NULL,
    retry_requested_at timestamp with time zone NOT NULL,
    pending_count bigint DEFAULT 0 NOT NULL,
    failed_count bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT olap_export_retry_commands_command_id_check CHECK ((command_id <> ''::text)),
    CONSTRAINT olap_export_retry_commands_failed_count_check CHECK ((failed_count >= 0)),
    CONSTRAINT olap_export_retry_commands_mode_check CHECK ((mode = ANY (ARRAY['retry_failed'::text, 'resume_from_checkpoint'::text]))),
    CONSTRAINT olap_export_retry_commands_pending_count_check CHECK ((pending_count >= 0)),
    CONSTRAINT olap_export_retry_commands_reason_check CHECK ((reason <> ''::text)),
    CONSTRAINT olap_export_retry_commands_stream_check CHECK ((stream = ANY (ARRAY['raw_business_events'::text, 'stock_moves'::text]))),
    CONSTRAINT olap_export_retry_commands_pkey PRIMARY KEY (command_id)
);

CREATE TABLE public.olap_operator_audit_events (
    id text NOT NULL,
    command_id text NOT NULL,
    action text NOT NULL,
    stream text NOT NULL,
    job_id text NOT NULL,
    requested_by text DEFAULT ''::text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT olap_operator_audit_events_action_check CHECK ((action = ANY (ARRAY['create_backfill_job'::text, 'cancel_backfill_job'::text]))),
    CONSTRAINT olap_operator_audit_events_command_id_check CHECK ((command_id <> ''::text)),
    CONSTRAINT olap_operator_audit_events_id_check CHECK ((id <> ''::text)),
    CONSTRAINT olap_operator_audit_events_stream_check CHECK ((stream = ANY (ARRAY['raw_business_events'::text, 'stock_moves'::text]))),
    CONSTRAINT olap_operator_audit_events_pkey PRIMARY KEY (id),
    CONSTRAINT olap_operator_audit_events_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.olap_backfill_jobs(id) ON DELETE RESTRICT
);

CREATE TABLE public.stock_documents (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    warehouse_id text,
    document_type text NOT NULL,
    source_event_id text NOT NULL,
    source_event_type text NOT NULL,
    business_date_local date NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT stock_documents_document_type_check CHECK ((document_type = ANY (ARRAY['SALE'::text, 'RETURN'::text, 'WASTE'::text, 'PRODUCTION'::text, 'PURCHASE'::text, 'ADJUSTMENT'::text, 'TRANSFER'::text, 'INVENTORY_COUNT'::text]))),
    CONSTRAINT stock_documents_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT stock_documents_source_event_id_check CHECK ((source_event_id <> ''::text)),
    CONSTRAINT stock_documents_source_event_type_check CHECK ((source_event_type <> ''::text)),
    CONSTRAINT stock_documents_pkey PRIMARY KEY (id)
);

CREATE TABLE public.inventory_document_processing_state (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    source_event_id text NOT NULL,
    source_event_type text NOT NULL,
    source_aggregate_id text,
    stock_document_id text,
    status text NOT NULL,
    posted_ledger_count bigint DEFAULT 0 NOT NULL,
    expected_ledger_count bigint,
    costing_status text NOT NULL,
    needs_recalculation boolean DEFAULT false NOT NULL,
    failure_code text,
    failure_message_key text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    posted_at timestamp with time zone,
    CONSTRAINT inventory_document_processing_state_costing_status_check CHECK ((costing_status = ANY (ARRAY['final'::text, 'estimated'::text, 'needs_recalculation'::text, 'recalculated'::text, 'failed'::text]))),
    CONSTRAINT inventory_document_processing_state_expected_ledger_count_check CHECK (((expected_ledger_count IS NULL) OR (expected_ledger_count >= 0))),
    CONSTRAINT inventory_document_processing_state_id_check CHECK ((id <> ''::text)),
    CONSTRAINT inventory_document_processing_state_posted_ledger_count_check CHECK ((posted_ledger_count >= 0)),
    CONSTRAINT inventory_document_processing_state_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT inventory_document_processing_state_source_event_id_check CHECK ((source_event_id <> ''::text)),
    CONSTRAINT inventory_document_processing_state_source_event_type_check CHECK ((source_event_type = ANY (ARRAY['StockReceiptCaptured'::text, 'InventoryCountCaptured'::text, 'StockWriteOffCaptured'::text, 'ProductionCompleted'::text]))),
    CONSTRAINT inventory_document_processing_state_status_check CHECK ((status = ANY (ARRAY['accepted'::text, 'posted'::text, 'partially_posted'::text, 'failed'::text, 'ignored_duplicate'::text]))),
    CONSTRAINT inventory_document_processing_restaurant_id_source_event_id_key UNIQUE (restaurant_id, source_event_id, source_event_type),
    CONSTRAINT inventory_document_processing_state_pkey PRIMARY KEY (id),
    CONSTRAINT inventory_document_processing_state_stock_document_id_fkey FOREIGN KEY (stock_document_id) REFERENCES public.stock_documents(id) ON DELETE RESTRICT
);

CREATE TABLE public.stock_ledger (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    warehouse_id text,
    stock_document_id text NOT NULL,
    source_event_id text NOT NULL,
    source_event_type text NOT NULL,
    catalog_item_id text NOT NULL,
    order_line_id text,
    movement_type text NOT NULL,
    quantity numeric(14,3) NOT NULL,
    unit_code text NOT NULL,
    unit_cost_minor bigint NOT NULL,
    total_cost_minor bigint NOT NULL,
    costing_status text NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    business_date_local date NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT stock_ledger_catalog_item_id_check CHECK ((catalog_item_id <> ''::text)),
    CONSTRAINT stock_ledger_costing_status_check CHECK ((costing_status = ANY (ARRAY['final'::text, 'estimated'::text, 'needs_recalculation'::text, 'recalculated'::text, 'failed'::text]))),
    CONSTRAINT stock_ledger_movement_type_check CHECK ((movement_type = ANY (ARRAY['IN'::text, 'OUT'::text]))),
    CONSTRAINT stock_ledger_quantity_check CHECK ((quantity > (0)::numeric)),
    CONSTRAINT stock_ledger_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT stock_ledger_source_event_id_check CHECK ((source_event_id <> ''::text)),
    CONSTRAINT stock_ledger_source_event_type_check CHECK ((source_event_type <> ''::text)),
    CONSTRAINT stock_ledger_unit_code_check CHECK ((unit_code <> ''::text)),
    CONSTRAINT stock_ledger_unit_cost_minor_check CHECK ((unit_cost_minor >= 0)),
    CONSTRAINT stock_ledger_pkey PRIMARY KEY (id),
    CONSTRAINT stock_ledger_stock_document_id_fkey FOREIGN KEY (stock_document_id) REFERENCES public.stock_documents(id) ON DELETE RESTRICT
);

CREATE TABLE public.inventory_stock_balances (
    restaurant_id text NOT NULL,
    warehouse_id text DEFAULT ''::text NOT NULL,
    catalog_item_id text NOT NULL,
    unit_code text NOT NULL,
    quantity_on_hand numeric(14,3) DEFAULT 0 NOT NULL,
    last_movement_at timestamp with time zone NOT NULL,
    last_ledger_entry_id text NOT NULL,
    costing_status text NOT NULL,
    needs_recalculation boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT inventory_stock_balances_catalog_item_id_check CHECK ((catalog_item_id <> ''::text)),
    CONSTRAINT inventory_stock_balances_costing_status_check CHECK ((costing_status = ANY (ARRAY['final'::text, 'estimated'::text, 'needs_recalculation'::text, 'recalculated'::text, 'failed'::text]))),
    CONSTRAINT inventory_stock_balances_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT inventory_stock_balances_unit_code_check CHECK ((unit_code <> ''::text)),
    CONSTRAINT inventory_stock_balances_pkey PRIMARY KEY (restaurant_id, warehouse_id, catalog_item_id, unit_code),
    CONSTRAINT inventory_stock_balances_last_ledger_entry_id_fkey FOREIGN KEY (last_ledger_entry_id) REFERENCES public.stock_ledger(id) ON DELETE RESTRICT
);

CREATE TABLE public.stock_recalculation_jobs (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    source_document_id text,
    trigger_type text NOT NULL,
    trigger_event_id text,
    trigger_command_id text,
    status text NOT NULL,
    business_date_from date NOT NULL,
    business_date_to date NOT NULL,
    affected_catalog_item_count integer DEFAULT 0 NOT NULL,
    affected_warehouse_count integer DEFAULT 0 NOT NULL,
    total_steps integer DEFAULT 0 NOT NULL,
    completed_steps integer DEFAULT 0 NOT NULL,
    failure_code text,
    failure_message_key text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT stock_recalculation_jobs_affected_catalog_item_count_check CHECK ((affected_catalog_item_count >= 0)),
    CONSTRAINT stock_recalculation_jobs_affected_warehouse_count_check CHECK ((affected_warehouse_count >= 0)),
    CONSTRAINT stock_recalculation_jobs_completed_steps_check CHECK ((completed_steps >= 0)),
    CONSTRAINT stock_recalculation_jobs_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT stock_recalculation_jobs_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'running'::text, 'completed'::text, 'failed'::text, 'cancelled'::text]))),
    CONSTRAINT stock_recalculation_jobs_total_steps_check CHECK ((total_steps >= 0)),
    CONSTRAINT stock_recalculation_jobs_trigger_type_check CHECK ((trigger_type <> ''::text)),
    CONSTRAINT stock_recalculation_jobs_pkey PRIMARY KEY (id),
    CONSTRAINT stock_recalculation_jobs_source_document_id_fkey FOREIGN KEY (source_document_id) REFERENCES public.stock_documents(id) ON DELETE RESTRICT
);

CREATE TABLE public.stock_recalculation_edges (
    job_id text NOT NULL,
    dependency_catalog_item_id text NOT NULL,
    dependent_catalog_item_id text NOT NULL,
    edge_type text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT stock_recalculation_edges_dependency_catalog_item_id_check CHECK ((dependency_catalog_item_id <> ''::text)),
    CONSTRAINT stock_recalculation_edges_dependent_catalog_item_id_check CHECK ((dependent_catalog_item_id <> ''::text)),
    CONSTRAINT stock_recalculation_edges_edge_type_check CHECK ((edge_type = ANY (ARRAY['recipe'::text, 'modifier_link'::text]))),
    CONSTRAINT stock_recalculation_edges_pkey PRIMARY KEY (job_id, dependency_catalog_item_id, dependent_catalog_item_id, edge_type),
    CONSTRAINT stock_recalculation_edges_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.stock_recalculation_jobs(id) ON DELETE CASCADE
);

CREATE TABLE public.stock_recalculation_job_items (
    job_id text NOT NULL,
    catalog_item_id text NOT NULL,
    warehouse_id text DEFAULT ''::text NOT NULL,
    unit_code text NOT NULL,
    business_date_from date NOT NULL,
    business_date_to date NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT stock_recalculation_job_items_catalog_item_id_check CHECK ((catalog_item_id <> ''::text)),
    CONSTRAINT stock_recalculation_job_items_unit_code_check CHECK ((unit_code <> ''::text)),
    CONSTRAINT stock_recalculation_job_items_pkey PRIMARY KEY (job_id, catalog_item_id, warehouse_id, unit_code),
    CONSTRAINT stock_recalculation_job_items_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.stock_recalculation_jobs(id) ON DELETE CASCADE
);

CREATE TABLE public.stop_lists (
    id text NOT NULL,
    restaurant_id text NOT NULL,
    catalog_item_id text NOT NULL,
    available_quantity numeric(14,3),
    source text NOT NULL,
    reason text,
    active boolean NOT NULL,
    cloud_version bigint,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT stop_lists_catalog_item_id_check CHECK ((catalog_item_id <> ''::text)),
    CONSTRAINT stop_lists_restaurant_id_check CHECK ((restaurant_id <> ''::text)),
    CONSTRAINT stop_lists_source_check CHECK ((source <> ''::text)),
    CONSTRAINT stop_lists_pkey PRIMARY KEY (id)
);

CREATE INDEX cloud_catalog_folders_parent_sort ON public.cloud_catalog_folders USING btree (restaurant_id, parent_id, sort_order, id);

CREATE UNIQUE INDEX cloud_catalog_items_active_sku ON public.cloud_catalog_items USING btree (sku) WHERE (status <> 'archived'::text);

CREATE INDEX cloud_catalog_items_restaurant_kind_status ON public.cloud_catalog_items USING btree (restaurant_id, kind, status);

CREATE INDEX cloud_categories_restaurant_status ON public.cloud_categories USING btree (restaurant_id, status, sort_order);

CREATE UNIQUE INDEX cloud_currency_reference_alpha_code_idx ON public.cloud_currency_reference USING btree (currency_alpha_code);

CREATE UNIQUE INDEX cloud_edge_event_receipts_edge_event_key ON public.cloud_edge_event_receipts USING btree (restaurant_id, device_id, edge_event_id);

CREATE INDEX cloud_edge_event_receipts_event_type_received_at ON public.cloud_edge_event_receipts USING btree (event_type, cloud_received_at);

CREATE INDEX cloud_edge_nodes_restaurant_status ON public.cloud_edge_nodes USING btree (restaurant_id, status);

CREATE INDEX cloud_employee_memberships_restaurant ON public.cloud_employee_restaurant_memberships USING btree (restaurant_id, employee_id);

CREATE INDEX cloud_employees_status ON public.cloud_employees USING btree (status);

CREATE UNIQUE INDEX cloud_halls_active_name ON public.cloud_halls USING btree (restaurant_id, name) WHERE (status <> 'archived'::text);

CREATE INDEX cloud_master_data_delivery_states_restaurant ON public.cloud_master_data_delivery_states USING btree (restaurant_id, status, updated_at DESC);

CREATE INDEX cloud_master_data_packages_stream_updated ON public.cloud_master_data_packages USING btree (stream_name, updated_at DESC);

CREATE INDEX cloud_master_data_publications_current ON public.cloud_master_data_publications USING btree (restaurant_id, version DESC) WHERE (status = 'published'::text);

CREATE INDEX cloud_menu_items_restaurant_status ON public.cloud_menu_items USING btree (restaurant_id, status);

CREATE UNIQUE INDEX cloud_operational_events_edge_event_key ON public.cloud_operational_events USING btree (restaurant_id, device_id, edge_event_id);

CREATE INDEX cloud_operational_events_restaurant_sequence ON public.cloud_operational_events USING btree (restaurant_id, device_id, occurred_at, event_id);

CREATE INDEX cloud_operational_events_type_received_at ON public.cloud_operational_events USING btree (event_type, cloud_received_at);

CREATE UNIQUE INDEX cloud_pairing_codes_one_active_per_restaurant ON public.cloud_pairing_codes USING btree (restaurant_id) WHERE (status = 'active'::text);

CREATE INDEX cloud_pairing_codes_restaurant_status ON public.cloud_pairing_codes USING btree (restaurant_id, status, expires_at);

CREATE INDEX cloud_pricing_policies_restaurant_active ON public.cloud_pricing_policies USING btree (restaurant_id, status, application_index);

CREATE INDEX cloud_printers_restaurant ON public.cloud_printers USING btree (restaurant_id, is_active);

CREATE INDEX cloud_restaurant_sections_restaurant_mode_active ON public.cloud_restaurant_sections USING btree (restaurant_id, mode, is_active, id);

CREATE UNIQUE INDEX cloud_restaurant_sections_one_default_hall_section ON public.cloud_restaurant_sections USING btree (restaurant_id) WHERE ((mode = 'hall_section'::text) AND (is_default = true));

CREATE INDEX cloud_sales_points_restaurant_active ON public.cloud_sales_points USING btree (restaurant_id, is_active, id);

CREATE INDEX cloud_projection_financial_operations_check ON public.cloud_projection_financial_operations USING btree (restaurant_id, check_id, operation_created_at DESC);

CREATE UNIQUE INDEX cloud_projection_financial_operations_edge_operation ON public.cloud_projection_financial_operations USING btree (restaurant_id, device_id, edge_operation_id);

CREATE INDEX cloud_projection_financial_operations_original_shift ON public.cloud_projection_financial_operations USING btree (restaurant_id, original_shift_id, operation_created_at DESC);

CREATE INDEX cloud_projection_financial_operations_restaurant_date_type ON public.cloud_projection_financial_operations USING btree (restaurant_id, business_date_local, operation_type, operation_created_at DESC);

CREATE INDEX cloud_projection_financial_operations_shift ON public.cloud_projection_financial_operations USING btree (restaurant_id, shift_id, operation_created_at DESC);

CREATE INDEX cloud_projection_stop_list_updates_action ON public.cloud_projection_stop_list_updates USING btree (projection_action, projected_at DESC);

CREATE INDEX cloud_projection_stop_list_updates_restaurant_updated ON public.cloud_projection_stop_list_updates USING btree (restaurant_id, updated_at DESC, source_event_id);

CREATE UNIQUE INDEX cloud_receipt_templates_default_uq ON public.cloud_receipt_templates USING btree (org_id, COALESCE(restaurant_id, ''::text), document_type) WHERE ((is_default = true) AND (is_active = true));

CREATE INDEX cloud_receipt_templates_org_type ON public.cloud_receipt_templates USING btree (org_id, document_type, is_active);

CREATE INDEX cloud_recipe_lines_version_order ON public.cloud_recipe_lines USING btree (recipe_version_id, sort_order, id);

CREATE UNIQUE INDEX cloud_recipe_versions_one_active ON public.cloud_recipe_versions USING btree (restaurant_id, owner_catalog_item_id) WHERE (status = 'active'::text);

CREATE INDEX cloud_recipe_versions_owner_status ON public.cloud_recipe_versions USING btree (restaurant_id, owner_catalog_item_id, status, version DESC);

CREATE INDEX cloud_restaurants_status_updated ON public.cloud_restaurants USING btree (status, updated_at DESC);

CREATE INDEX cloud_review_assignment_audit_events_review_created ON public.cloud_review_assignment_audit_events USING btree (review_type, review_id, occurred_at DESC);

CREATE UNIQUE INDEX cloud_roles_name_unique ON public.cloud_roles USING btree (name);

CREATE INDEX cloud_sync_problem_events_created_at ON public.cloud_sync_problem_events USING btree (created_at DESC);

CREATE UNIQUE INDEX cloud_tables_active_name ON public.cloud_tables USING btree (section_id, name) WHERE (status <> 'archived'::text);

CREATE INDEX cloud_tables_restaurant_section ON public.cloud_tables USING btree (restaurant_id, section_id);

CREATE UNIQUE INDEX cloud_tables_one_default_per_restaurant ON public.cloud_tables USING btree (restaurant_id) WHERE (is_default = true);

CREATE INDEX cloud_unassigned_edge_nodes_status_seen ON public.cloud_unassigned_edge_nodes USING btree (status, last_seen_at DESC);

CREATE INDEX inbox_events_event_type_received ON public.inbox_events USING btree (event_type, cloud_received_at DESC, id DESC);

CREATE UNIQUE INDEX inbox_events_event_unique ON public.inbox_events USING btree (restaurant_id, device_id, event_id);

CREATE INDEX inbox_events_olap_pending ON public.inbox_events USING btree (processed_for_olap, olap_export_status, olap_next_retry_at, cloud_received_at, id);

CREATE INDEX inbox_events_restaurant_received ON public.inbox_events USING btree (restaurant_id, cloud_received_at DESC, id DESC);

CREATE INDEX inventory_document_processing_state_document ON public.inventory_document_processing_state USING btree (stock_document_id) WHERE (stock_document_id IS NOT NULL);

CREATE INDEX inventory_document_processing_state_restaurant_type_status ON public.inventory_document_processing_state USING btree (restaurant_id, source_event_type, status, updated_at DESC, id DESC);

CREATE UNIQUE INDEX inventory_document_processing_state_source_event_unique ON public.inventory_document_processing_state USING btree (restaurant_id, source_event_id, source_event_type);

CREATE INDEX inventory_event_queue_event_type ON public.inventory_event_queue USING btree (event_type, occurred_at, id);

CREATE INDEX inventory_event_queue_restaurant_warehouse_order ON public.inventory_event_queue USING btree (restaurant_id, warehouse_id, occurred_at, id);

CREATE INDEX inventory_event_queue_status_retry ON public.inventory_event_queue USING btree (status, next_retry_at, occurred_at, id);

CREATE INDEX inventory_stock_balances_costing_status ON public.inventory_stock_balances USING btree (restaurant_id, costing_status, last_movement_at DESC);

CREATE INDEX inventory_stock_balances_restaurant_last_movement ON public.inventory_stock_balances USING btree (restaurant_id, last_movement_at DESC, catalog_item_id, unit_code);

CREATE INDEX inventory_stock_balances_restaurant_warehouse_item ON public.inventory_stock_balances USING btree (restaurant_id, warehouse_id, catalog_item_id);

CREATE INDEX olap_backfill_jobs_stream_status_created ON public.olap_backfill_jobs USING btree (stream, status, created_at DESC);

CREATE INDEX olap_export_retry_commands_stream_created ON public.olap_export_retry_commands USING btree (stream, created_at DESC);

CREATE INDEX olap_operator_audit_events_job_created ON public.olap_operator_audit_events USING btree (job_id, created_at DESC);

CREATE INDEX stock_documents_restaurant_occurred_at ON public.stock_documents USING btree (restaurant_id, occurred_at, id);

CREATE INDEX stock_documents_restaurant_warehouse_occurred_at ON public.stock_documents USING btree (restaurant_id, warehouse_id, occurred_at, id);

CREATE UNIQUE INDEX stock_documents_source_event_unique ON public.stock_documents USING btree (source_event_id, source_event_type);

CREATE INDEX stock_ledger_order_line_consumption ON public.stock_ledger USING btree (restaurant_id, order_line_id, source_event_type, movement_type);

CREATE INDEX stock_ledger_restaurant_occurred_at ON public.stock_ledger USING btree (restaurant_id, occurred_at, id);

CREATE INDEX stock_ledger_restaurant_warehouse_occurred_at ON public.stock_ledger USING btree (restaurant_id, warehouse_id, occurred_at, id);

CREATE INDEX stock_ledger_source_event ON public.stock_ledger USING btree (source_event_id, source_event_type);

CREATE INDEX stock_recalculation_edges_job_order ON public.stock_recalculation_edges USING btree (job_id, sort_order, dependency_catalog_item_id, dependent_catalog_item_id);

CREATE INDEX stock_recalculation_job_items_item ON public.stock_recalculation_job_items USING btree (catalog_item_id, warehouse_id, business_date_from, business_date_to);

CREATE INDEX stock_recalculation_jobs_restaurant_status ON public.stock_recalculation_jobs USING btree (restaurant_id, status, created_at, id);

CREATE UNIQUE INDEX stock_recalculation_jobs_trigger_command_unique ON public.stock_recalculation_jobs USING btree (restaurant_id, trigger_type, trigger_command_id) WHERE (trigger_command_id IS NOT NULL);

CREATE UNIQUE INDEX stock_recalculation_jobs_trigger_event_unique ON public.stock_recalculation_jobs USING btree (restaurant_id, trigger_type, trigger_event_id) WHERE (trigger_event_id IS NOT NULL);

CREATE UNIQUE INDEX stop_lists_restaurant_item ON public.stop_lists USING btree (restaurant_id, catalog_item_id);
