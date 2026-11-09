# AGENTS.md

Guidance for agents (and humans) that implement or review changes in the Infisical Terraform provider.

Read this file before you write code or review a pull request. It applies to every resource, data source, ephemeral resource and client change in this repository.

## The rules that matter most

1. **The API must be ready before it goes into Terraform.** When a resource ships, its API shape is locked into user configurations and state files. We cannot change it later without a new API version. Audit the Infisical API against the [API acceptance criteria](#api-acceptance-criteria) first. If the API fails a criterion, do not build the resource and do not change Infisical core on your own. Write an [API change proposal](#if-the-api-fails-a-criterion) and wait until a person accepts it.
2. **Mirror the API, the UI and the docs.** Read the route, the request and response schemas, the service and the UI form in Infisical core before you design the schema. Put each field in the same place that the API puts it.
3. **Use the base resources.** Secret syncs, app connections, secret rotations, dynamic secrets, secret validation rules, certificate syncs, external KMS and PAM accounts share a base implementation. A field that the API defines for all variants goes into the base, not into one variant. If you find a new group of resources with one shared API shape, create a new base.
4. **Every resource must be Crossplane-compatible by default.** Follow the [Read rules](#crossplane-compatibility). Do not create Crossplane-specific copies of resources.
5. **Generated docs, examples and tests are part of the change.** Run `go generate ./...` and commit the result.

## Repository map

| Path | What it contains |
| --- | --- |
| `main.go` | Provider entry point and the `go:generate` directives (format examples, generate docs, set doc subcategories). |
| `internal/provider/provider.go` | Provider schema, authentication, and the registration lists for resources, data sources and ephemeral resources. |
| `internal/provider/resource/` | Resources. Families with a shared base live in sub-packages (see [Base resources](#base-resources)). |
| `internal/provider/datasource/` | Data sources. |
| `internal/client/` | The HTTP client for the Infisical API (resty). One file per API area. Request and response structs are in `model.go` or in the area file. |
| `internal/errors/` | `APIError` and `GenericRequestError`. Use these for every client error. |
| `internal/pkg/validators/` | Reusable schema validators (`MsDurationBetween`, `RegexPattern`, `BoolTrueOrNull`, `ObjectAtLeastOneAttributeSet`). |
| `internal/pkg/modifiers/` | Reusable plan modifiers (JSON equivalence, unordered lists, trim equality). |
| `internal/pkg/customtypes/` | Custom attribute types with semantic equality (`TrimmedString`, `KubernetesHost`). |
| `internal/pkg/terraform/` | Shared conversion helpers (`Int64PtrIfKnown`, `OptionalStringPointer`, `UuidValidator`, role helpers). |
| `examples/` | Example HCL that `tfplugindocs` puts into the docs. |
| `templates/` | Custom doc templates for pages that need more than the default layout. |
| `docs/` | **Generated.** Do not edit by hand. |
| `scripts/update-doc-subcategories.sh` | Sets the Terraform Registry sidebar category for each doc page. |

## Commands

```sh
go build ./...                     # compile
go test ./...                      # unit tests (no Infisical instance needed)
golangci-lint run                  # lint (CI uses golangci-lint v2.15.0 with .golangci.yml)
go generate ./...                  # format examples, regenerate docs/, set doc subcategories
make testacc                       # TF_ACC=1 go test ./... (see step 5)
```

`go generate` needs the `terraform` CLI on your `PATH`, because it runs `terraform fmt -recursive ./examples/`.

CI (`.github/workflows/test.yml`) runs a build, the linter, a `go generate` diff check and the tests. The diff check fails if `docs/` or `examples/` change after `go generate`. Always run it and commit the output.

To try a local build with a real Terraform configuration, add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "infisical/infisical" = "/path/to/your/GOPATH/bin"
  }
  direct {}
}
```

Then run `go install` and run `terraform plan` in a directory that uses the provider. You do not need `terraform init` for a provider with a dev override.

## How to do a task

### 1. Collect context

- If the task has a Linear issue, read the issue and all of its comments.
- Find the matching code in Infisical core. The core repository is usually a sibling of this one (`../infisical`). If you do not have it, ask for it. Do not guess API shapes from the docs or from memory.
- Read the `AGENTS.md` files in Infisical core before you search it (see [step 2](#2-audit-the-api-before-you-build)).
- Find the newest sibling resource in this repository that does a similar thing. Use it as your template. Old resources often have patterns that we no longer want.

### 2. Audit the API before you build

When a resource ships in Terraform, users write its attributes into their configurations and Terraform writes the API responses into state. After that, a change to the API shape breaks users. The only safe way to change it is a new API version. So the API must be correct **before** we build on it. Features often ship in the UI with an API that did not get enough attention. Find those problems now, not after release.

**Where to look in Infisical core.** Infisical core describes its own architecture in `AGENTS.md` files: one at the root, one in `backend/` and in `frontend/`, and more in module directories. The core architecture changes over time, so this file does not list core paths. Read the core `AGENTS.md` files first, then find these items for the feature:

- The routes: method, path, and the request and response schemas.
- The shared route factory and shared schemas, when the feature is a family (for example, one set of routes for every secret sync destination).
- The service code. Many rules (permission checks, conflicts, rules across fields) live there and not in the request schema.
- The API field descriptions. They are a good source for Terraform attribute descriptions.
- The errors that the routes return, and their status codes.
- The UI form for the feature, and the API types that the UI uses.
- The public docs for the feature.
- The API design rules that core defines (the core `AGENTS.md` files point to them). Audit the API against those rules and against the criteria below.

#### API acceptance criteria

The API is in good shape when it meets every criterion below. Check each one and write the result into the pull request description.

- [ ] **Full lifecycle.** There is a create, a get by ID, an update and a delete endpoint. The get by ID endpoint returns **404** when the object does not exist (not 400, 403 or 500).
- [ ] **Round trip.** The get response returns every field that create and update accept, in the same shape and with the same names. If a field does not come back, Terraform cannot detect drift or import it. Secret values are the exception: for those, the API must return something that changes when the secret changes (for example `credentialsHash` on app connections).
- [ ] **Import from ID only.** The get response contains everything Terraform needs to build full state from the ID alone, including parent IDs such as `projectId` and the connection ID.
- [ ] **Correct field placement.** A field that applies to every variant of a family is in the shared base schema of the API, not in one variant. A field that is specific to a destination or provider is in that variant's schema. Field names are the same across variants. The field is in the group where it belongs (for example, an option about which secrets to read from Infisical is a source option, not a destination option). The UI shows the field in the same group.
- [ ] **Consistent naming.** New endpoints use `projectId` (not `workspaceId`), camelCase keys, and the same names that sibling endpoints use for the same concept.
- [ ] **Update semantics are clear.** We know what happens when a field is omitted from a PATCH, and what happens when it is sent as `null`. Terraform must be able to clear an optional field and set a boolean to `false`. We know which fields PATCH cannot change, so they can use `RequiresReplace()`. We know whether PATCH replaces a nested object as a whole or merges it.
- [ ] **No silent changes to values.** The API does not trim, lowercase, reorder or fill in values without a documented rule. If it does, the provider must handle it with a custom type, a plan modifier or a computed attribute. Lists come back in a stable order.
- [ ] **Defaults are explicit.** Server-side defaults are in the request schema, so the provider can use the same default.
- [ ] **Clear validation.** Enums are explicit. Validation errors tell the user what is wrong.
- [ ] **Machine identities can call it.** The provider authenticates as a machine identity. The endpoints must accept machine identity tokens and the permission checks must work for them.
- [ ] **Public and current.** The route does not set `hide: true` (hidden routes are internal and can change at any time). The route is not deprecated, and there is no newer version of it.
- [ ] **UI parity.** The UI form and the API expose the same options. If the UI has an option that the API does not have (or the opposite), find out why.

#### If the API fails a criterion

Do not build the Terraform resource, and do not work around the problem in the provider. A workaround becomes permanent behavior that users depend on. Do not change Infisical core on your own either. Write an API change proposal and give it to the person who asked for the task. The proposal contains:

1. **The criterion that fails**, from the list above.
2. **The evidence:** the route, the schema or service code in core, and an example request or response that shows the problem.
3. **The effect on Terraform:** what users cannot do, or what we lock in, if we build on the API as it is.
4. **The proposed change:** the API shape before and after. Keep the change backward compatible when you can, for example with the old field kept as a deprecated mirror.
5. **Open questions** that a person must answer.

A person must accept the proposal before anyone changes the API. The Terraform work starts after the accepted API change is released.

### 3. Design the Terraform schema

- **Resource names:** `infisical_<family>_<variant>`, for example `infisical_secret_sync_aws_secrets_manager` or `infisical_app_connection_spacelift`. Use the API slug of the variant, in snake_case.
- **Attribute names:** snake_case of the API key (`includeAllSubFolders` becomes `include_all_sub_folders`). Keep the API's nesting: if the API has `syncOptions.keySchema`, the attribute is `sync_options.key_schema`.
- **Required, Optional, Computed:**
  - `Required` when the API requires it.
  - `Optional` + `Computed` + `Default` when the API has a default. Use the **same** default as the API.
  - `Computed` with `UseStateForUnknown()` for server-generated values that do not change, such as `id`.
  - `RequiresReplace()` for fields that the update endpoint does not accept (for example `project_id`).
- **IDs:** use `UuidValidator` (in `internal/pkg/terraform`) only for IDs that the API defines as UUIDs. Do **not** validate `project_id` as a UUID: the API types project IDs as plain strings, and old projects do not have UUIDs.
- **Secrets:** mark every secret `Sensitive: true`. For new secret inputs, also consider a write-only attribute (`WriteOnly: true`) with a `<name>_wo_version` trigger, as in `internal/provider/resource/secret_resource.go`. Write-only attributes need Terraform 1.11 or later. Say so in the description.
- **Validate at plan time.** Use validators (`stringvalidator.OneOf`, `ConflictsWith`, `AlsoRequires`, the validators in `internal/pkg/validators`) so that the user gets an error at `plan`, not a 400 at `apply`. Use `ValidateConfig` for rules across attributes. Do not copy all of the API validation. Validate what users can easily get wrong.
- **Descriptions:** every attribute has a `Description`. List the allowed values and the default. The docs come from these descriptions.
- **New fields and older servers.** Self-hosted users can run an older Infisical version than Infisical Cloud. An older server ignores fields that it does not know, and it does not return an error. If an attribute needs a recent Infisical version, give the minimum version in the description.
- **No breaking changes by default.** Do not rename or remove attributes, change their types, or make an optional attribute required. Add a `DeprecationMessage` and keep the old attribute working. If a schema change makes old state unreadable, add a schema `Version` and an `UpgradeState` implementation. A breaking change needs an explicit decision from the team (see the [`include_all_sub_folders` example](#example-include_all_sub_folders)). When the team approves one, the release notes must tell users what to change.

### 4. Implement

**Client (`internal/client/`)**

- One function per endpoint. Use the existing pattern: `R().SetResult(&body).SetHeader("User-Agent", USER_AGENT)`, an `operation...` constant for the error, `errors.NewGenericRequestError` for transport errors and `errors.NewAPIErrorWithResponse` for API errors.
- On every get by ID, return `ErrNotFound` when the status is 404. Resources depend on this.
- Keep Terraform types out of the client. The client uses plain Go types and JSON tags.
- **Be careful with `omitempty`.** On an update request, `omitempty` on a `bool` or a number drops `false` and `0`, so the user cannot set those values. Use a pointer when "not set" and "zero" mean different things, and send the field without `omitempty` when the zero value is valid. Send an explicit `null` when the API needs it to clear a field (see `gatewayIdForRequest` in `base_app_connection.go`).
- The client retries 429 responses (see `retry.go`). Do not add your own retry loop for rate limits.

**Resource**

- Add compile-time interface checks (`var _ resource.ResourceWithImportState = &MyResource{}`).
- `Configure` stores the `*infisical.Client`. Copy the pattern from an existing resource.
- Check `r.client.Config.IsMachineIdentityAuth` at the start of each CRUD method, like the other resources.
- `Create` and `Update` send the plan, then write the plan to state with the server-generated values filled in.
- `Read` follows the [Crossplane rules](#crossplane-compatibility) and sets **every** attribute from the API response, so drift is visible. Keep null and empty values consistent: if the user did not set an optional string and the API returns `""`, keep it null (see how the base resources handle `description`).
- `Delete` treats a 404 as success. The object is already gone, which is the result that the delete wanted.
- `ImportState` accepts the ID, checks that it is a UUID when the API uses UUIDs, and passes it through with `resource.ImportStatePassthroughID`. `Read` then fills in the state. Do not write a separate code path for import.
- Register the constructor in `internal/provider/provider.go`.

**Docs and examples**

- Add `examples/resources/<resource_name>/resource.tf` and `import.sh`. For data sources, add `examples/data-sources/<name>/data-source.tf`. Use placeholders like `<your-infisical-project-id>`. Copy the provider block from a sibling example.
- If you add a new family or a new name prefix, add a case for it in `scripts/update-doc-subcategories.sh`.
- Use a template in `templates/` only when the generated page is not enough.
- Run `go generate ./...` and commit the changes in `docs/`.

### 5. Test

There are no acceptance tests against a live Infisical instance in CI. The tests are unit tests that drive the resource against a fake API:

- Client tests use `testClient` and `jsonResponse` in `internal/client/testutil_test.go`.
- Resource lifecycle tests use an `httptest` fake backend. Examples: `internal/provider/resource/pam/pam_fake_backend_test.go` and `internal/provider/resource/sub_organization_group_lifecycle_test.go`. Make the fake behave like the real API, including its errors and quirks, and say which API code you copied the behavior from.
- `internal/provider/provider_schema_test.go` runs the framework's schema validation over all resources. It must pass.

Write tests for: the request body that the resource sends (field names and placement), the mapping from the API response to state, a 404 on read (state is removed), an empty ID on read, and import. When you change a base resource, test it with more than one variant.

When you can, also test manually against a real instance with a dev override. Run `plan` again after `apply` and confirm that it shows no changes. Then import the resource into an empty state and confirm that `plan` shows no changes.

## Base resources

Most of the provider is families of resources that share one API shape. Each family has a base type that holds the schema, the CRUD logic and the API calls. Each variant only supplies its own attributes and the functions that convert between the Terraform model and the API payload.

| Family | Base | A variant supplies |
| --- | --- | --- |
| Secret syncs | `secret_sync/base_secret_sync.go` (`SecretSyncBaseResource`) | `App`, `SyncName`, `AppConnection`, `CanImportSecrets`, `DestinationConfigAttributes`, `SyncOptionsAttributes`, and the `Read...FromPlan` and `Read...FromApi` functions for both |
| App connections | `app_connection/base_app_connection.go` (`AppConnectionBaseResource`) | `App`, `AllowedMethods`, `CredentialsAttributes`, credential read functions, `OverwriteCredentialsFields`, `SupportsGateway` and `RequiresGateway`, optional `IsRetryableError` |
| Secret rotations | `secret_rotation/base_secret_rotation.go` (`SecretRotationBaseResource`) | `Provider`, `AppConnection`, `ParametersAttributes`, `SecretsMappingAttributes`, `TemporaryParametersAttributes`, and their read functions |
| Dynamic secrets | `dynamic_secret/base_dynamic_secret.go` (`DynamicSecretBaseResource`) | `Provider`, `ConfigurationAttributes`, `ReadConfigurationFromPlan`, `ReadConfigurationFromApi` |
| Secret validation rules | `secret_validation_rule/base_secret_validation_rule.go` (`SecretValidationRuleBaseResource`) | `Type`, `ConstraintsAttributes`, `ReadConstraintsFromPlan`, `ReadConstraintsFromApi` |
| Certificate syncs | `certificate_sync/base_certificate_sync.go` (`CertificateSyncBaseResource`) | Destination and sync option attributes and their conversions |
| External KMS | `external_kms/base_external_kms.go` (`ExternalKmsBaseResource`) | Provider-specific configuration |
| PAM | `pam/base_pam_account.go`, `pam/base_pam_membership.go` | Account type specifics, membership scope |

All paths are relative to `internal/provider/resource/`.

### Rules for base resources

- **A field that the API defines for every variant goes into the base.** Add it to the base model, the base schema, the create and update requests, and `Read`. Do not add it to one variant. If you add it to one variant, every other variant becomes inconsistent, and someone must find and fix them later.
- **A field that only some variants support goes behind a capability flag on the base**, like `SupportsGateway` or `CanImportSecrets`. The API often has the same kind of flag (for example, which shared sync options a destination supports, or whether an app connection supports gateways). Use the same split. Do not copy the same attribute code into many variant files. If the same code would appear in two or more variant files, move it into the base or into a shared helper in the same package.
- **Before you change a base, read all of its variants.** A base change affects every variant at the same time. Check that each variant still builds the correct request, run all tests, and run `go generate ./...` so that the docs of every variant update.
- **Make a new base when you see a new family.** If you implement two or more resources that share one API shape (one set of routes in Infisical core with a variant segment in the path), build a base type first and make each resource a small variant. Abstraction wins here: it gives consistent behavior, docs and fixes across many resources.
- **A new variant copies the newest variant**, not the oldest.

### API facts per family

These facts describe API behavior in Infisical core. Check them again in the code before you depend on them, because the API can change.

- **Secret syncs.** The API defines the shared fields and the shared sync options once, for all destinations. A destination adds its own `destinationConfig` and can add extra sync options. Each destination has flags that say which shared sync options it supports (importing secrets, key schema, disabling secret deletion). For most destinations, PATCH replaces `syncOptions` as a whole, so always send the full object.
- **App connections.** Responses never contain secret credentials. The provider uses `credentialsHash` to detect a change to the credentials outside of Terraform (see `Read` in the base). `projectId` is optional: without it, the connection belongs to the organization.
- **Secret rotations.** PATCH cannot change `connectionId`, `environment` or `secretPath`. A rotation moves with a separate endpoint.
- **Dynamic secrets.** This API is older. It finds a dynamic secret by name, project slug, environment slug and path, not by ID, and PATCH nests changes under `data`. Read responses hide secret fields, and an update that omits them keeps the stored values. A new API with this shape would not pass the audit in step 2.

### Example: `include_all_sub_folders`

**What went wrong.** A change added `include_all_sub_folders` to `sync_options` of one of the 18 secret sync resources (AWS Secrets Manager). It was reverted (PR #392, reverted in #394). Spacelift already had its own copy of the attribute, also in `sync_options`. Nobody audited the API first. An audit would have found two problems:

1. The API accepts the field for every destination. So it belongs in the base secret sync resource, not in one variant.
2. The field selects which secrets to read from Infisical. That makes it a source option, like `environment` and `secretPath`, not a sync option. The UI showed it with the source fields, but the API put it in `syncOptions`.

**The fix, in order:**

1. **Change the API first.** The team accepted a proposal to move the field. Infisical core moves `includeAllSubFolders` to the top level of the sync, next to `environment` and `secretPath`. GET returns it at the top level. If an update leaves it out, the API keeps the current value. The API still accepts and returns `syncOptions.includeAllSubFolders` as a deprecated mirror, so released provider versions keep working. When both are set, the top-level value wins.
2. **Then change Terraform, after the API change is released.**
   - The base secret sync resource gets one top-level `include_all_sub_folders` attribute (optional, default `false`) for every destination.
   - The base sends and reads the top-level API field, never the mirror. It always sends the value, because an update that leaves the field out cannot set it back to `false`.
   - The API rejects `true` unless `initialSyncBehavior` is `overwrite-destination`. A `ValidateConfig` rule in the base gives this error at plan time, for destinations that expose `initial_sync_behavior`.
   - Spacelift's `sync_options.include_all_sub_folders` is removed. This is a breaking change that the team decided on. The release notes must tell Spacelift users to move the attribute to the top level.

Before you add a field to one variant of a family:

1. Find where the field is defined in Infisical core. Is it in the shared schema of the family, or in the schema of one variant? Is it at the top level of the body, in `syncOptions`, or in `destinationConfig`? Where does the UI show it?
2. If it is shared, add it to the base resource once, in the same position as in the API.
3. If the API placement is wrong or inconsistent, write an [API change proposal](#if-the-api-fails-a-criterion).

## Crossplane compatibility

The provider is also used through Crossplane. A resource that follows the rules below works in Crossplane with no extra code. This is required for every new resource and for every resource you change.

In `Read`:

1. If `state.ID` is empty, call `resp.State.RemoveResource(ctx)` and return.
2. If the API returns 404 (`infisical.ErrNotFound` from the client), call `resp.State.RemoveResource(ctx)` and return. Do **not** return an error.

When the resource is removed from state, Terraform (and Crossplane) runs `Create` again. Use `Read` in `internal/provider/resource/identity.go` as the reference:

```go
if state.ID.ValueString() == "" {
	resp.State.RemoveResource(ctx)
	return
}

orgIdentity, err := r.client.GetIdentity(infisical.GetIdentityRequest{
	IdentityID: state.ID.ValueString(),
})

if err != nil {
	if err == infisical.ErrNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.AddError("Error reading identity", "Couldn't read identity from Infisical, unexpected error: "+err.Error())
	return
}
```

Many older resources check for 404 but not for an empty ID. When you change one of them, add the empty ID check.

Crossplane uses the normal provider release (the `v*` tags). There is no separate Crossplane build, so never write a Crossplane-specific version of a resource. Make the normal resource follow the rules above.

## Code style

- The Go version and dependencies come from `go.mod`. The provider uses `terraform-plugin-framework` (not SDKv2).
- Match the surrounding code: naming, comment density, error message wording, and `any` versus `interface{}`.
- Error diagnostics use the existing wording: summary `"Error creating <thing>"`, detail `"Couldn't create <thing>, unexpected error: " + err.Error()`.
- The linter config (`.golangci.yml`) enables, among others, `godot` (comments end with a period), `errcheck`, `forcetypeassert` (use `v, ok := x.(T)`), `unparam` and `unused`.
- Comments explain **why**, not what. Explain API quirks where you handle them, and name the API behavior that you handle.
- Use the helpers in `internal/pkg/` before you write a new one. Put a new helper there when more than one package needs it.
- In this file and in other agent docs, refer to symbols and file paths, not line numbers. Line numbers go out of date.

## Review checklist

A reviewer (human or agent) checks these items. The implementer checks them before asking for review.

- [ ] The PR describes the API audit: the endpoints, the Infisical core files that were read, and the result of each [API acceptance criterion](#api-acceptance-criteria). For each failed criterion, a person accepted an API change proposal and the API change is released, or the PR explains why the gap is acceptable.
- [ ] Each attribute matches the API: name, nesting, type, required or optional, and default. Compare with the request and response schemas, not only with the docs.
- [ ] The UI and the API agree on the options that the resource exposes, and on where each option belongs.
- [ ] Shared fields are in the base resource. No attribute code is copied across variants. When a base changed, all variants were checked.
- [ ] Update requests can send `false`, `0` and `null` where the user can set them (no `omitempty` traps). Fields that PATCH does not accept use `RequiresReplace()`.
- [ ] `Read` removes the resource from state on 404 and on an empty ID. `Read` sets every attribute from the API.
- [ ] `Delete` treats 404 as success.
- [ ] Import works from the ID alone and produces no diff on the next plan.
- [ ] Secret inputs are `Sensitive`. Secret values do not appear in logs or diagnostics.
- [ ] Enums and simple rules are validated at plan time. `project_id` is not validated as a UUID.
- [ ] No breaking schema changes without deprecation (and a state upgrade where needed), unless the team decided on the break. A decided break has release notes that tell users what to change.
- [ ] Examples (`resource.tf`, `import.sh`) exist, `go generate ./...` was run, and `docs/` is committed. New families have a subcategory in `scripts/update-doc-subcategories.sh`.
- [ ] Tests cover the request body, the API-to-state mapping, 404 on read, an empty ID on read, and import.
