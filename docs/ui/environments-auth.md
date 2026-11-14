# Environments and Authentication Layout

Configuration surfaces for variable scopes/environments and request authentication.

## Environments & Variables

### Layout

```text
+------------------------------------------------------------------+
| Environment selector / actions (create, duplicate, delete, color)|
+------------------------------------------------------------------+
| Variable table/editor                                            |
|  Name | Value / Expression | Type/Flags | Secret | Description   |
+------------------------------------------------------------------+
| Scope/inheritance hints (workspace/collection/folder/request)    |
+------------------------------------------------------------------+
```

### Responsibilities

- Manage multiple environments per workspace (e.g. dev/staging/prod) and active selection.
- Edit variables with names, values/template expressions, secret designation, enablement, and optional descriptions.
- Show effective/inherited values and precedence without leaking secret plaintext.
- Support public/shareable vs private/secret data distinctions aligned with sync/export policies.

### Behavior

- Template-aware value inputs with validation and resolution feedback.
- Prevent accidental exposure: mask secrets by default, explicit reveal/copy actions, no logging.
- Color/label affordances for quick environment recognition.

## Authentication

### Layout

```text
+------------------------------------------------------------------+
| Auth type selector (inherit / none / basic / bearer / api key /  |
| oauth2 / aws / custom provider...)                               |
+------------------------------------------------------------------+
| Dynamic fields for selected provider                             |
| (credentials, tokens, scopes, region/service, headers/params)    |
+------------------------------------------------------------------+
| Inheritance/source indicator and test/refresh actions (if any)   |
+------------------------------------------------------------------+
```

### Responsibilities

- Configure auth at request, folder/collection, or workspace scope with clear inheritance/override behavior.
- Render provider-specific fields (including advanced OAuth/OIDC/AWS flows) through schema-driven/dynamic forms.
- Separate secret credentials from non-sensitive config and route storage through secure services.

### Behavior

- "Inherit" uses nearest defined parent auth; "None" explicitly disables.
- OAuth flows may require interactive steps/token refresh status; expose progress/errors without blocking unrelated UI.
- Changes affect resolved outbound requests only after save/apply according to draft policy.

## Yaak Reference Mapping

- `EnvironmentEditor.tsx`, `EnvironmentEditDialog.tsx`, `CreateEnvironmentDialog.tsx`, `EnvironmentActionsDropdown.tsx`, `EnvironmentColorPicker.tsx`, `HttpAuthenticationEditor.tsx`, `DynamicForm.tsx`.
- Heartless should implement native/core auth providers and environment resolution services first; plugin-driven providers can follow via isolated adapters.
