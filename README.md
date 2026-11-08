# Members CRUD demo for OpenChoreo

A small HTTP REST backend for the export-engine presentation. Written in Go using
only the standard library: no framework, database, package downloads, or gRPC.
It starts with Alice and Bob so the first list request has something to show.

The runtime image contains only a static binary and runs as a non-root user.
Source builds still need access to the Go builder container image.

## Start locally

Requires Go 1.22 or newer (the Dockerfile uses Go 1.27.3).

```sh
cd ~/Desktop/members-demo-api
go run .
```

Keep that terminal open. The service listens on port 8080. Test in another terminal:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/members
```

If port 8080 is occupied, use `PORT=8081 go run .` and change the Postman local
environment's `base_url` to `http://localhost:8081`.

To run the automated checks:

```sh
go test -race ./...
go vet ./...
```

Optional local container build:

```sh
docker build -t members-demo-api:demo .
docker run --rm -p 8080:8080 members-demo-api:demo
```

## API contract

Set `base_url` to `http://localhost:8080` locally, or the deployed endpoint's full
invoke base URL in OpenChoreo. Do not add a trailing slash or `/members` to it.

| Action | Method | Path | Success |
| --- | --- | --- | --- |
| Health | GET | `/healthz` | 200 |
| Show all members | GET | `/members` | 200 |
| Add a member | POST | `/members` | 201 |
| Show one member | GET | `/members/{id}` | 200 |
| Edit a member | PUT | `/members/{id}` | 200 |
| Delete a member | DELETE | `/members/{id}` | 204, empty body |

POST and PUT require `Content-Type: application/json`. Both require `name` and
a valid `email`. `role` is optional and defaults to `member`. PUT replaces the
fields; send the complete name/email/role you want to keep. IDs are generated
by the server. The `role` field is a display value and grants no permissions.

Add request:

```json
{
  "name": "Charlie Fernando",
  "email": "charlie@example.com",
  "role": "member"
}
```

Response (ID 3 on a fresh instance):

```json
{
  "data": {
    "id": 3,
    "name": "Charlie Fernando",
    "email": "charlie@example.com",
    "role": "member"
  }
}
```

Edit request:

```json
{
  "name": "Charlie Fernando Updated",
  "email": "charlie.updated@example.com",
  "role": "admin"
}
```

GET `/members` returns `{"data": [...], "count": 2}` initially. A missing
member returns HTTP 404 with `{"error":"member not found"}`. Bad fields,
invalid IDs or malformed JSON return 400; wrong Content-Type returns 415;
bodies over 64 KiB return 413. Unsupported methods return 405.

## Postman: import the ready-made collection

1. Open the Postman desktop app and your workspace.
2. Use Import (or Use resources or import > Import), or drag in these files:
   - `postman/members-crud.postman_collection.json`
   - `postman/local.postman_environment.json`
   - `postman/openchoreo.postman_environment.json`
3. Select the `Members API - Local` environment for the locally running backend.
4. Open `OpenChoreo - Members CRUD Demo` under Collections.
5. Send requests 01 through 09 in order. Each has response assertions under
   Scripts > Post-response and results in the response test-results panel.
6. Request 03 automatically saves the response's `data.id` as the collection
   variable `member_id`. Later requests use it, so there is no manual ID copying.
7. To run the entire sequence automatically, open the collection's Run action,
   select all nine requests in their saved order, and use one iteration.

| Order | Saved request | What the audience sees |
| --- | --- | --- |
| 01 | Health check | `status: ok` |
| 02 | Show all members | Alice and Bob on a fresh process |
| 03 | Add member | A new member with a generated ID |
| 04 | Show created member | The newly added member |
| 05 | Edit member | Updated name, email, and role |
| 06 | Verify edited member | A fresh GET confirms the edit |
| 07 | Delete member | HTTP 204 with an empty response body |
| 08 | Verify deleted member returns 404 | Expected HTTP 404; its test passes |
| 09 | Show members after delete | The member is absent from the list |

To use OpenChoreo, select `Members API - OpenChoreo`, then edit its `base_url`
to the exact invoke base URL shown by your deployed component's HTTP endpoint.
Its initial value is only the usual local quick-start example:

```text
http://development-default.openchoreoapis.localhost:19080/members-api-http
```

Keep any gateway prefix in that URL. The collection appends `/members` or
`/healthz` automatically. Do not point deployed requests at `localhost:8080`:
that would exercise the local process. If using Postman in a browser, select
the Desktop Agent to reach a localhost cluster.

Do not create an environment variable named `member_id`; the collection owns
that variable. Create again before repeating get/edit/delete requests after a
backend restart. Run the sequence on one instance and use one replica.

## Postman: create the collection manually, if preferred

1. Under Collections, create a collection called `Members Demo`.
2. In its Variables tab, add `base_url = http://localhost:8080` and an empty
   `member_id` variable. Save the collection.
3. Add each request from the API table. Use URLs such as `{{base_url}}/members`
   and `{{base_url}}/members/{{member_id}}`.
4. For Add and Edit, select Body > raw > JSON, paste the corresponding example,
   and ensure the Content-Type header is `application/json`.
5. In Add member > Scripts > Post-response, save the returned ID:

```javascript
pm.test("Created", function () {
  pm.response.to.have.status(201);
});
if (pm.response.code === 201) {
  pm.collectionVariables.set("member_id", String(pm.response.json().data.id));
}
```

6. Save the requests and send Add before Show one, Edit, or Delete. Switch the
   base URL to the deployed invoke URL when demonstrating OpenChoreo.

An OpenAPI 3.0 specification is also included as `openapi.json`; Postman can
import it to generate requests, but the prepared collection already contains
the demo ordering, payloads, ID capture, and assertions.

## Build and deploy from source in OpenChoreo

This follows the source layout and `dockerfile-builder` configuration in the
OpenChoreo checkout used to prepare this project. It assumes a configured
`default` namespace, default deployment pipeline and project type, the
`deployment/service` component type, a working `dockerfile-builder` workflow,
registry access, and a ready development environment.

### 1. Put this folder in an accessible Git repository

Create an empty repository named `members-demo-api` in your Git provider. Put
the contents of this folder at its root. Before the initial commit, replace
`YOUR_USERNAME` in `openchoreo/component.yaml` with the repository owner.

If using GitHub, the commands are:

```sh
cd ~/Desktop/members-demo-api
git init -b main
git add .
git commit -m "Add lightweight members CRUD demo"
git remote add origin https://github.com/YOUR_USERNAME/members-demo-api.git
git push -u origin main
```

Replace `YOUR_USERNAME` in the command as well. No remote repository has been
created or published automatically. A public demo repository avoids Git
credential configuration; for a private repository configure the platform's
repository credentials. The cluster must be able to fetch the source.

### 2. Create through the portal (presentation route)

1. Select the `default` namespace and create the `export-demo` project using
   the default project type/pipeline, or open it if you already created it.
   Ensure it is deployed to development and its project binding is ready.
2. Add a component named `members-api` from a Git repository.
3. Select the service component type (`deployment/service`) and the Dockerfile
   build workflow (`dockerfile-builder`). Configure:

| Field | Value |
| --- | --- |
| Repository | `https://github.com/YOUR_USERNAME/members-demo-api` |
| Branch | `main` |
| Application path | `.` |
| Build context | `.` |
| Dockerfile path | `./Dockerfile` |
| Automatic deployment | Enabled |

4. Start the component's build workflow. The source-root `workload.yaml`
   declares the external HTTP endpoint on port 8080 and includes the API schema.
   The workflow generates the actual Workload with the built image.
5. Wait for the build and development deployment to be ready. Copy the HTTP
   invoke base URL into the Postman OpenChoreo environment.

Portal labels may differ by installed version. The supplied manifests express
the exact equivalent source configuration for this checkout.

### 3. Equivalent CLI route (use instead of duplicate portal creation)

After replacing the repository URL in `openchoreo/component.yaml`:

```sh
occ login
occ apply -f openchoreo/project.yaml
```

If the project already exists through the portal, skip applying that project
manifest to preserve its existing settings. Ensure its development project
binding exists and is ready. Then:

```sh
occ apply -f openchoreo/component.yaml
occ component workflow run members-api --namespace default --project export-demo
occ component workflowrun list members-api --namespace default --project export-demo
occ component workflowrun logs members-api --namespace default --project export-demo
```

With `autoDeploy: true`, the new workload should deploy automatically after a
successful workflow. Inspect the component in the portal and wait for its
development release binding to be ready before using Postman.

If your setup uses another namespace, change both manifests and every
`--namespace` argument consistently. If you use a monorepo subfolder, change
repository `appPath`, Docker context and Dockerfile path to that subfolder.

The root `workload.yaml` is a source descriptor, NOT an apply-ready Workload CR.
Apply only the two files under `openchoreo/` for this source deployment route.

## Use in the export presentation

Source builds involve checkout, image building, publishing, and deployment.
Rehearse that duration; a tiny backend does not make the entire workflow
instant. To keep your 10-12 minute demo on time, begin the build early and
explain the CLI while it runs, or prepare a built-image deployment for the
quick live creation step. No image has been pushed by this setup.

Suggested live sequence:

1. Create/deploy `export-demo` / `members-api` through the portal.
2. In Postman, send Health, Show all, and Add member. Keep its captured ID.
3. Run `occ export -o ./demo-gitops`.
4. Show the exported project, component, workload, releases, and bindings under
   `namespaces/default/projects/export-demo/`.
5. Return to Postman and send Show created, Edit, Verify edited, Delete, and
   Verify deleted. This shows application behavior after export.
6. For credential detection, add the unused dummy environment variable
   `DEMO_API_TOKEN=dummy-demo-value` through the portal workload configuration
   and export with `--show-credential-details`. The backend ignores that variable.

Export captures OpenChoreo configuration, not the member records stored by this
application. Adding the dummy environment variable may roll out a new process;
finish the CRUD sequence first because member data resets on restart.

## Demo boundaries

Data lives in memory and resets to Alice and Bob on process restart or redeploy.
Use one replica so successive requests see the same member list. There is no
authentication: this backend is intended for demo data in a demo environment.
Its member role field is separate from OpenChoreo RBAC.

## Files

- `main.go`: backend implementation.
- `main_test.go`: CRUD lifecycle, input validation, and concurrent request checks.
- `Dockerfile`: multi-stage source build and minimal non-root runtime image.
- `workload.yaml`: OpenChoreo source workload descriptor, external HTTP port 8080.
- `openapi.json`: endpoint specification and request/response schemas.
- `openchoreo/`: project/development binding and source component manifests.
- `postman/`: prepared collection and local/deployed environments.

Reference: [Postman import guide](https://learning.postman.com/docs/getting-started/importing-and-exporting/importing-data/),
[Postman environment variables](https://learning.postman.com/docs/use/send-requests/variables/environment-variables/).
