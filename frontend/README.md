# Members frontend

A small member directory with search, add, edit, and remove actions. Plain
HTML/CSS/JavaScript is embedded in a Go server; no Node.js or npm is required.
The server forwards `/api/*` to the existing members backend, so browser
requests use the frontend's origin. The API does not need CORS changes.

## Run locally

Start the backend from the repository root:

```sh
go run .
```

In another terminal, from the repository root:

```sh
cd frontend
go run .
```

Open **http://localhost:8081**. The frontend defaults to port 8081 and connects
to `http://localhost:8080`. Override these at runtime if needed:

```sh
PORT=8082 API_BASE_URL=http://localhost:8080 go run .
```

`API_BASE_URL` is the backend's base URL, without `/members` or `/api`.
It can include a gateway prefix. For example, to use your deployed backend:

```sh
API_BASE_URL=http://development-default.openchoreoapis.localhost:19080/members-api-http go run .
```

Changes to the static files require restarting `go run .` because they are
embedded in the binary. Data is stored by the backend and resets on its restart.

## Create the second component in OpenChoreo

Commit and push the new files to the same GitHub repository before building.
In the existing **export-demo** project, add a component with these settings:

| Field | Value |
| --- | --- |
| Component name | `members-frontend` |
| Component type | `deployment/service` |
| Build workflow | `dockerfile-builder` |
| Repository | `https://github.com/sakith03/members-demo-api.git` |
| Branch | `main` |
| Application path | `frontend` |
| Docker build context | `frontend` |
| Dockerfile path | `frontend/Dockerfile` |
| Automatic deployment | Enabled |

The Docker context and Dockerfile path are relative to the **repository root**.
The workflow reads `frontend/workload.yaml` using the application path.
It exposes HTTP on port 8080 and sets `API_BASE_URL=http://members-api:8080`.
This uses the backend's Kubernetes Service directly. Put both components in
the same project and environment; adjust the URL if the backend service is renamed.

The equivalent component manifest is `../openchoreo/frontend-component.yaml`.
Use this instead of creating a duplicate through the portal:

```sh
# From the repository root, after pushing the files:
occ apply -f openchoreo/frontend-component.yaml
occ component workflow run members-frontend --namespace default --project export-demo
```

Once ready, open the frontend HTTP invoke URL shown in the portal. In the
current local setup it should be:

```text
http://development-default.openchoreoapis.localhost:19080/members-frontend-http
```

The page supports the gateway path with or without a trailing slash.
`frontend/workload.yaml` is a source descriptor, not an apply-ready Kubernetes CR.

## Container and checks

From the repository root:

```sh
docker build -t members-demo-frontend:demo frontend
# Connect to the API running on the host (Docker Desktop / Colima):
docker run --rm -p 8081:8080 \
  -e API_BASE_URL=http://host.docker.internal:8080 \
  members-demo-frontend:demo
```

Use a host address reachable from the container if your Docker installation
does not provide `host.docker.internal`.

From `frontend/`:

```sh
go test -race ./...
go vet ./...
```

The frontend is a separate Go module; run its checks separately from the root
backend module. `/healthz` checks the frontend process; the directory's API
connection indicator reflects successful member-list requests.
