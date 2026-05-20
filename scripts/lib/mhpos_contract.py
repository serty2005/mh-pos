import json
import pathlib
import urllib.parse


DEFAULT_SPEC = pathlib.Path(__file__).resolve().parents[2] / "docs" / "api" / "mhpos-local-smoke.openapi.json"
HTTP_METHODS = {"get", "post", "put", "patch", "delete"}


class OpenAPIContract:
    def __init__(self, spec):
        self.spec = spec
        self.operations = {}
        for path, path_item in spec.get("paths", {}).items():
            for method, operation in path_item.items():
                if method.lower() not in HTTP_METHODS:
                    continue
                operation_id = operation.get("operationId")
                if not operation_id:
                    continue
                if operation_id in self.operations:
                    raise ValueError(f"duplicate operationId {operation_id}")
                self.operations[operation_id] = (method.upper(), path, operation)

    def operation(self, operation_id):
        try:
            return self.operations[operation_id]
        except KeyError as exc:
            raise KeyError(f"unknown OpenAPI operationId {operation_id}") from exc

    def build_request(self, operation_id, path_params=None, query=None, body=None):
        method, path, operation = self.operation(operation_id)
        self._validate_body(operation_id, operation, body)
        return {
            "method": method,
            "path": self._build_path(path, path_params or {}, query or {}),
            "api_prefix": bool(operation.get("x-mhpos-api-prefix", True)),
            "expected_status": self._expected_statuses(operation),
        }

    def _build_path(self, path, path_params, query):
        out = path
        for name, value in path_params.items():
            token = "{" + name + "}"
            if token not in out:
                continue
            out = out.replace(token, urllib.parse.quote(str(value), safe=""))
        if "{" in out or "}" in out:
            raise ValueError(f"missing path parameter for {path}")
        clean_query = {key: value for key, value in query.items() if value is not None and value != ""}
        if clean_query:
            out += "?" + urllib.parse.urlencode(clean_query)
        return out

    def _validate_body(self, operation_id, operation, body):
        schema = self._request_schema(operation)
        if schema is None:
            return
        required = tuple(schema.get("required", ()))
        if not required:
            return
        if body is None:
            raise ValueError(f"{operation_id} requires request body fields: {', '.join(required)}")
        missing = [name for name in required if name not in body]
        if missing:
            raise ValueError(f"{operation_id} missing required request body fields: {', '.join(missing)}")

    def _request_schema(self, operation):
        content = operation.get("requestBody", {}).get("content", {})
        schema = content.get("application/json", {}).get("schema")
        if not schema:
            return None
        return self._resolve_schema(schema)

    def _resolve_schema(self, schema):
        ref = schema.get("$ref")
        if not ref:
            return schema
        prefix = "#/components/schemas/"
        if not ref.startswith(prefix):
            raise ValueError(f"unsupported schema reference {ref}")
        name = ref[len(prefix) :]
        try:
            return self.spec["components"]["schemas"][name]
        except KeyError as exc:
            raise KeyError(f"unknown schema reference {ref}") from exc

    def _expected_statuses(self, operation):
        statuses = []
        for raw in operation.get("responses", {}):
            if raw.isdigit():
                code = int(raw)
                if 200 <= code < 300:
                    statuses.append(code)
        return tuple(sorted(statuses)) or (200,)


def load_contract(path):
    with open(path, "r", encoding="utf-8") as fh:
        return OpenAPIContract(json.load(fh))


def load_default_contract():
    return load_contract(DEFAULT_SPEC)
