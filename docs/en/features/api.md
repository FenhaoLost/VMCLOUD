# API Integration

CLICD remains compatible with the legacy `/api` endpoints, so existing integrations do not need changes. For new integrations, use the `/api/v1` endpoints; the list below covers v1. `GET /api/v1/containers` is recommended for listing containers.

## Authentication

API keys can be created and managed on the "API Integration" page. Two header styles are supported when making requests:

```bash
curl -H "X-API-Key: YOUR_API_KEY" https://panel.example.com/api/v1/containers
```

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" https://panel.example.com/api/v1/dashboard
```

## Response Structure

All endpoints return a uniform response wrapper:

```json
{
  "success": true,
  "message": "OK",
  "data": {}
}
```

When integrating, only read the fields your business needs. New capabilities are added as optional fields first, so existing plugins never have to rename their current fields.

## Creating and Reinstalling

Creating containers, batch creation, reinstalling, and batch reinstalling now support mixed NAT/public IPv4/IPv6 networking and Linux SSH login configuration. The public IPv4/IPv6 address pools can be viewed via `GET /api/v1/routing` and updated via `PUT /api/v1/routing`.

Example: create a container

```json
{
  "name": "demo-lxc-01",
  "virtualization": "lxc",
  "template_id": "debian-bookworm",
  "vcpu": 1,
  "ram_mb": 512,
  "disk_gb": 10,
  "assign_nat": true,
  "port_mapping_count": 2,
  "assign_ipv4": false,
  "ipv4_count": 1,
  "public_ipv4s": [],
  "assign_ipv6": true,
  "ipv6_count": 1,
  "ipv6_addresses": [],
  "ssh_auth_mode": "auto_password",
  "ssh_password": "",
  "ssh_public_key": "",
  "expires_at": "",
  "network_down_mbps": 100,
  "network_up_mbps": 50,
  "io_read_mbps": 120,
  "io_write_mbps": 80
}
```

Field reference:

| Field | Description |
| --- | --- |
| `assign_nat` | Whether to allocate NAT port mappings; when omitted, the default NAT behavior is kept. |
| `assign_ipv4` | Whether to assign public IPv4. |
| `ipv4_count` | Number of public IPv4 addresses to auto-assign. |
| `public_ipv4s` | Explicit list of public IPv4 addresses. |
| `assign_ipv6` | Whether to assign IPv6. |
| `ipv6_count` | Number of IPv6 addresses to auto-assign. |
| `ipv6_addresses` | Explicit list of IPv6 addresses. |
| `ssh_auth_mode` | Linux creation supports `auto_password`, `password`, `key`; reinstall additionally supports `keep`. |
| `ssh_password` | Custom password in `password` mode; 8-64 characters, must contain both letters and digits, and no whitespace. |
| `ssh_public_key` | A single-line SSH public key in `key` mode. |
| `network_down_mbps` | Optional; container download/ingress bandwidth limit in Mbps, `0` means unlimited. |
| `network_up_mbps` | Optional; container upload/egress bandwidth limit in Mbps, `0` means unlimited. |
| `io_read_mbps` | Optional; disk read rate limit in MB/s, `0` means unlimited. |
| `io_write_mbps` | Optional; disk write rate limit in MB/s, `0` means unlimited. |
| `network_bw_mbps` | Legacy field; sets symmetric download/upload bandwidth at once. New integrations should use the split fields. |
| `io_speed_mbps` | Legacy field; sets symmetric read/write I/O limits at once. New integrations should use the split fields. |

Example: reinstall

```json
{
  "template_id": "debian-bookworm",
  "ssh_auth_mode": "keep",
  "ssh_password": "",
  "ssh_public_key": ""
}
```

`keep` is only for reinstall and means to reuse the current SSH password. Windows KVM images ignore the Linux SSH public key fields.

## Resource and Traffic Limits

`PUT /api/v1/containers/{id}/resource-limit` supports partial updates per field; omitted fields are left unchanged.

```json
{
  "vcpu": 2,
  "ram_mb": 1024,
  "network_down_mbps": 100,
  "network_up_mbps": 50,
  "io_read_mbps": 120,
  "io_write_mbps": 80
}
```

The legacy `network_bw_mbps` and `io_speed_mbps` fields remain available, meaning symmetric download/upload bandwidth and symmetric read/write I/O limits respectively. For new integrations, use the split fields to control download/upload and read/write independently.

Request body for `PUT /api/v1/containers/{id}/traffic-limit`:

```json
{
  "traffic_mode": "total",
  "monthly_traffic_gb": 1024,
  "traffic_in_gb": 0,
  "traffic_out_gb": 0
}
```

| Field | Description |
| --- | --- |
| `traffic_mode` | Traffic limit mode; `total` applies a combined limit, `split` limits inbound/outbound separately. |
| `monthly_traffic_gb` | Monthly total traffic quota in `total` mode, in GB; `0` means unlimited. |
| `traffic_in_gb` | Monthly inbound quota in `split` mode, in GB; `0` means unlimited. |
| `traffic_out_gb` | Monthly outbound quota in `split` mode, in GB; `0` means unlimited. |

## Container Firewall

The container firewall is read via `GET /api/v1/containers/{id}/firewall` and updated via `PUT /api/v1/containers/{id}/firewall`. Updates are applied to running containers immediately.

Example update:

```json
{
  "enabled": true,
  "default_action": "DROP",
  "rules": [
    {
      "direction": "in",
      "protocol": "tcp",
      "action": "ACCEPT",
      "network": "ipv4",
      "source_ip": "203.0.113.0/24",
      "port": "22,80,443",
      "description": "allow admin and web"
    }
  ]
}
```

| Field | Description |
| --- | --- |
| `enabled` | Whether the container firewall is enabled. |
| `default_action` | Default action: `ACCEPT` or `DROP`. |
| `rules[].id` | Optional; may be omitted for new rules, the backend generates it automatically. |
| `rules[].direction` | Direction: `in` or `out`. |
| `rules[].protocol` | Protocol: `tcp`, `udp`, `icmp`, or `all`. |
| `rules[].action` | Action: `ACCEPT` or `DROP`. |
| `rules[].network` | Network type: `ipv4`, `ipv6`, or `all`. |
| `rules[].source_ip` | Optional; source IP, CIDR, or address range. |
| `rules[].port` | Optional; only for `tcp`/`udp`; accepts `22`, `80,443`, or `8000-9000`. |
| `rules[].description` | Optional remark. |

## API Key Creation and Update

`POST /api/v1/api-keys` and `PATCH /api/v1/api-keys/{id}` use the same field structure. `name` is required on create; override fields as needed on update.

```json
{
  "name": "Automation",
  "ip_whitelist": "198.51.100.23,203.0.113.0/24",
  "scopes": ["dashboard:read", "container:read", "container:power"],
  "expires_at": "2026-12-31 23:59:59",
  "disabled": false,
  "container_uuids": ["00000000-0000-4000-8000-000000000005"]
}
```

| Field | Description |
| --- | --- |
| `name` | API key name; required on create. |
| `ip_whitelist` | Optional; allowed source IP/CIDR, comma-separated; empty means no restriction. |
| `scopes` | Optional; permission scopes. Omitted to use the default read-only scopes, pass `*` for all permissions. |
| `expires_at` | Optional; expiration time, empty means no expiry. |
| `disabled` | Whether the key is disabled. |
| `container_uuids` | Optional; restricts the key to the specified containers only. |

## Panel Access Source Policy

`GET /api/v1/access-policy` reads the panel access allowlist, and `PUT /api/v1/access-policy` updates the policy. Both require the `admin:access` scope. The policy covers the panel pages, the login endpoint, and all APIs.

```json
{
  "enabled": true,
  "allowed_sources": [
    "203.0.113.10",
    "192.168.1.0/24",
    "2001:db8::/32"
  ],
  "trusted_proxies": [
    "127.0.0.1"
  ]
}
```

Both `allowed_sources` and `trusted_proxies` support IPv4, IPv6, and CIDR. The backend only honors `X-Forwarded-For`, `X-Real-IP`, or `CF-Connecting-IP` when the direct connection source matches `trusted_proxies`; spoofed headers from other clients cannot bypass the allowlist. At least one allowed source must be configured when the policy is enabled, and the endpoint rejects configurations that would exclude the current admin's source. Local loopback direct access remains as a recovery channel for CLI/SSH troubleshooting.

## Python Examples

Get the container list:

```python
import requests

BASE_URL = "https://panel.example.com"
API_KEY = "YOUR_API_KEY"

session = requests.Session()
session.headers.update({
    "X-API-Key": API_KEY,
    "Content-Type": "application/json",
})

resp = session.get(f"{BASE_URL}/api/v1/containers", timeout=15)
resp.raise_for_status()
print(resp.json())
```

Create a port mapping:

```python
import requests

BASE_URL = "https://panel.example.com"
API_KEY = "YOUR_API_KEY"
CONTAINER_ID = "example-vm"

payload = {
    "protocol": "tcp",
    "host_port": 18080,
    "container_port": 80,
    "description": "web",
}

resp = requests.post(
    f"{BASE_URL}/api/v1/containers/{CONTAINER_ID}/port-mappings",
    headers={"X-API-Key": API_KEY},
    json=payload,
    timeout=15,
)
resp.raise_for_status()
print(resp.json())
```

## Endpoint Reference

### Overview

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/dashboard` | Dashboard statistics |
| GET | `/api/v1/host-info` | Host resources |
| GET | `/api/v1/host-report` | Host inspection report |
| GET | `/api/v1/routing` | NAT/IPv4/IPv6 routing |
| PUT | `/api/v1/routing` | Update public IPv4/IPv6 pools |
| POST | `/api/v1/routing/ipv4-scan` | Scan public IPv4 ranges |
| GET | `/api/v1/ipv6/status` | IPv6 status |
| GET | `/api/v1/tasks` | Task queue |
| DELETE | `/api/v1/tasks/{task_id}` | Delete a task |

### Containers

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/containers` | Container list (recommended) |
| GET | `/api/v1/containers/list` | Container list (GET form) |
| POST | `/api/v1/containers/list` | Container list (POST form) |
| POST | `/api/v1/containers` | Create a container |
| GET | `/api/v1/containers/{id\|uuid\|name}` | Container details |
| POST | `/api/v1/containers/{id}/start` | Start |
| POST | `/api/v1/containers/{id}/stop` | Stop |
| POST | `/api/v1/containers/{id}/restart` | Restart |
| POST | `/api/v1/containers/{id}/reinstall` | Reinstall |
| DELETE | `/api/v1/containers/{id}/delete` | Delete |
| GET | `/api/v1/containers/{id}/usage` | Resource usage |
| GET | `/api/v1/containers/{id}/traffic` | Traffic statistics |
| POST | `/api/v1/containers/{id}/traffic-reset` | Reset traffic |
| PUT | `/api/v1/containers/{id}/traffic-limit` | Update traffic limits |
| PUT | `/api/v1/containers/{id}/resource-limit` | Update resource limits |
| PUT | `/api/v1/containers/{id}/expiry` | Update expiration time |
| POST | `/api/v1/containers/{id}/reset-password` | Reset SSH password |
| POST | `/api/v1/containers/{id}/ipv6` | Assign IPv6 |

### Ports and Snapshots

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/containers/{id}/random-port` | Random available port; pass `host_ip` to query a specific host IP |
| POST | `/api/v1/containers/{id}/port-mappings` | Add a port mapping |
| PUT | `/api/v1/containers/{id}/port-mappings/{index}` | Update a port mapping |
| DELETE | `/api/v1/containers/{id}/port-mappings/{index}` | Delete a port mapping |
| GET | `/api/v1/containers/{id}/firewall` | Get container firewall settings |
| PUT | `/api/v1/containers/{id}/firewall` | Update container firewall settings |
| GET | `/api/v1/snapshots` | Snapshot overview |
| GET | `/api/v1/containers/{id}/snapshots` | Container snapshots |
| POST | `/api/v1/containers/{id}/snapshots` | Create a snapshot |
| DELETE | `/api/v1/containers/{id}/snapshots/{snapshot_id}` | Delete a snapshot |
| POST | `/api/v1/containers/{id}/snapshots/{snapshot_id}/restore` | Restore a snapshot |
| POST | `/api/v1/containers/{id}/snapshots/schedule` | Scheduled snapshots |
| PUT | `/api/v1/containers/{id}/snapshots/quota` | Snapshot quota |

### Platform Management

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/templates` | Template list |
| GET | `/api/v1/images` | Image management list |
| GET | `/api/v1/images/enabled` | Enabled and downloaded images; supports `type=lxc\|kvm` |
| POST | `/api/v1/images/custom` | Add a third-party LXC/KVM image source |
| DELETE | `/api/v1/images/custom` | Remove a third-party LXC/KVM image source and its cache |
| POST | `/api/v1/images/download` | Download an image |
| POST | `/api/v1/images/cancel` | Cancel an image download |
| DELETE | `/api/v1/images/delete` | Delete image cache |
| PUT | `/api/v1/images/toggle` | Enable/disable an image |
| GET | `/api/v1/security/alerts` | Security alerts |
| POST | `/api/v1/security/check` | Run a security check now |
| GET | `/api/v1/security/logs?container={name}` | Security connection logs |
| GET | `/api/v1/security/summary` | Security summary |
| GET | `/api/v1/security/settings` | Security settings |
| PUT | `/api/v1/security/settings` | Update security settings |
| GET | `/api/v1/swap` | Swap info |
| POST | `/api/v1/swap` | Adjust Swap |
| GET | `/api/v1/language` | Current panel language |
| POST/PUT | `/api/v1/language` | Update panel language |
| GET | `/api/v1/ssl` | SSL settings (requires admin / `admin:access`) |
| PUT | `/api/v1/ssl` | Update SSL settings (requires admin / `admin:access`) |
| GET | `/api/v1/webssh-origins` | WebSSH Origin allowlist (requires admin / `admin:access`) |
| PUT | `/api/v1/webssh-origins` | Update WebSSH Origin allowlist (requires admin / `admin:access`) |
| POST | `/api/v1/batch-create` | Batch create containers |
| POST | `/api/v1/batch-action` | Batch start/stop/delete/reinstall |
| POST | `/api/v1/ssh-ticket` | Create a WebSSH ticket |
| POST | `/api/v1/vnc-ticket` | Create a WebVNC ticket |

### Storage / Policies / Migration

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/storage` | Storage pools and disk status |
| GET | `/api/v1/policies` | Policy list and trigger history |
| POST | `/api/v1/policies` | Create a policy |
| PUT | `/api/v1/policies/{id}` | Update a policy |
| DELETE | `/api/v1/policies/{id}` | Delete a policy |
| POST | `/api/v1/migrate/import` | Import a migration bundle |

### Node Management (Controller-Agent)

Controller-Agent endpoints use the `/api/nodes` prefix and require an administrator login session (JWT) or a node token:

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/nodes` | Node list (admin) |
| POST | `/api/nodes` | Create a node (admin) |
| DELETE | `/api/nodes/{id}` | Delete a node (admin) |
| GET | `/api/nodes/{id}` | Node details (admin) |
| GET | `/api/nodes/{id}/install-script` | One-line install script (admin) |
| GET | `/api/nodes/binary` | Download the Controller binary |
| POST | `/api/nodes/register` | Worker registration (install key) |
| POST | `/api/nodes/{id}/heartbeat` | Worker heartbeat (node token) |
| GET | `/api/nodes/{id}/containers` | Proxy view of worker containers (admin) |
| POST | `/api/nodes/{id}/containers/{cid}/{action}` | Proxy operation on worker containers (admin) |

Worker-side endpoints (`/api/agent/*`) only accept the Controller node token and are used for the Controller to proxy views and operations:

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/agent/containers` | Worker container list |
| POST | `/api/agent/containers/{cid}/{action}` | Start/stop/restart a worker container |
| POST | `/api/agent/action` | Compatible container action endpoint |

Example: create a node

```json
{
  "name": "node-1",
  "address": "http://203.0.113.20:8999"
}
```

Example: worker registration

```json
{
  "install_key": "安装密钥",
  "name": "node-1",
  "address": "http://203.0.113.20:8999",
  "version": "1.1.29"
}
```

On successful registration the worker receives its `node_id` and `token`; the worker uses the token to report heartbeats, and the Controller uses it to proxy access to the worker.

### Accounts and Logs

| Method | Path | Description |
| --- | --- | --- |
| POST | `/api/v1/sub-user/create` | Create a sub-user link |
| GET | `/api/v1/sub-users` | Sub-user list |
| POST | `/api/v1/sub-users/{id}/rotate-password` | Rotate a sub-user password |
| GET | `/api/v1/sub-users/{id}/audit-logs` | Sub-user operation logs |
| GET | `/api/v1/sub-users/{id}/login-logs` | Sub-user login logs |
| GET | `/api/v1/audit-logs` | Operation logs |
| GET | `/api/v1/login-logs` | Login logs |
| GET | `/api/v1/api-keys` | API key list |
| POST | `/api/v1/api-keys` | Create an API key |
| PATCH | `/api/v1/api-keys/{id}` | Update an API key |
| DELETE | `/api/v1/api-keys/{id}` | Delete an API key |

## Response Examples

The samples below are grouped by endpoint path. Real resource values, task IDs, container IDs, times, IPs, and keys differ in production; the passwords, tickets, and API keys in the examples have been masked.

### Overview

```json
{
  "GET /api/v1/dashboard": {
    "success": true,
    "data": {
      "running": 31,
      "stopped": 0,
      "total_containers": 31
    }
  },
  "GET /api/v1/host-info": {
    "success": true,
    "data": {
      "cpu": { "cores": 8, "usage_pct": 1.16 },
      "ram": { "total_mb": 31825, "used_mb": 1275, "free_mb": 30550 },
      "disk": { "total_gb": 1750.49, "used_gb": 123.98, "free_gb": 1626.51 },
      "network": {
        "public_ipv4": "203.0.113.10",
        "public_ipv4_interface": "eth0",
        "public_ipv6": "2001:db8:100::2",
        "public_ipv6_interface": "eth0"
      },
      "load": { "load1": 0.01, "load5": 0.03, "load15": 0.01 }
    }
  },
  "GET /api/v1/host-report": {
    "success": true,
    "data": {
      "generated_at": "2026-06-12 10:00:00",
      "summary": { "status": "ok", "warnings": 0 },
      "host": { "hostname": "node-1", "kernel": "6.8.0" },
      "resources": { "cpu_cores": 8, "ram_total_mb": 31825, "disk_total_gb": 1750.49 },
      "network": { "public_ipv4": "203.0.113.10", "public_ipv6": "2001:db8:100::2" }
    }
  },
  "GET /api/v1/routing": {
    "success": true,
    "data": {
      "nat4": { "used": 62, "remaining": "45474", "total": "45536" },
      "ipv4": { "used": 1, "remaining": "3", "total": "4" },
      "ipv6": { "used": 31, "remaining": "large", "total": "large" },
      "public_ipv4_addresses": [
        { "address": "203.0.113.10", "interface": "eth0", "prefix_len": 32, "gateway": "203.0.113.1" }
      ],
      "ipv4_assignments": [
        { "container_id": 5, "container_name": "example-vm", "address": "203.0.113.10", "interface": "eth0", "prefix_len": 32, "gateway": "203.0.113.1" }
      ],
      "nat4_mappings": [
        { "container_id": 5, "container_name": "example-vm", "status": "running", "ip": "10.0.0.10", "host_port": 22004, "container_port": 22, "protocol": "tcp" }
      ],
      "ipv6_assignments": [
        { "container_id": 5, "container_name": "example-vm", "address": "2001:db8:100::1005", "prefix_len": 64, "interface": "eth0" }
      ]
    }
  },
  "PUT /api/v1/routing": {
    "success": true,
    "data": {
      "ipv4": { "used": 1, "remaining": "3", "total": "4" },
      "public_ipv4_addresses": [
        { "address": "203.0.113.10", "interface": "eth0", "prefix_len": 32, "gateway": "203.0.113.1" }
      ],
      "ipv6_prefixes": [
        { "interface": "eth0", "address": "2001:db8:100::2", "prefix": "2001:db8:100::/64", "prefix_len": 64, "gateway": "2001:db8:100::1" }
      ]
    }
  },
  "POST /api/v1/routing/ipv4-scan": {
    "success": true,
    "data": [
      { "address": "203.0.113.10", "interface": "eth0", "prefix_len": 32, "gateway": "203.0.113.1", "status": "available", "usable": true, "reason": "" }
    ]
  },
  "GET /api/v1/ipv6/status": {
    "success": true,
    "data": {
      "available": true,
      "reachable": true,
      "reason": "usable public IPv6 prefix detected",
      "prefixes": [
        { "interface": "eth0", "address": "2001:db8:100::2", "prefix": "2001:db8:100::/64", "prefix_len": 64, "gateway": "2001:db8:100::1" }
      ]
    }
  },
  "GET /api/v1/tasks": {
    "success": true,
    "data": []
  },
  "DELETE /api/v1/tasks/{task_id}": {
    "success": true,
    "message": "Task deleted"
  }
}
```

### Containers

```json
{
  "GET /api/v1/containers": {
    "success": true,
    "data": [
      {
        "id": 5,
        "uuid": "00000000-0000-4000-8000-000000000005",
        "name": "example-vm",
        "virtualization": "lxc",
        "template": "debian-bullseye",
        "vcpu": 1,
        "ram_mb": 512,
        "disk_gb": 10,
        "network_down_mbps": 100,
        "network_up_mbps": 50,
        "io_read_mbps": 120,
        "io_write_mbps": 80,
        "status": "running",
        "ip": "10.0.0.10",
        "ipv6": "2001:db8:100::1005",
        "ssh_port": 22004,
        "ssh_password": "***",
        "port_mappings": [
          { "container_port": 22, "host_port": 22004, "protocol": "tcp", "description": "SSH" },
          { "container_port": 20000, "host_port": 20000, "protocol": "tcp", "description": "Port-20000" }
        ]
      }
    ]
  },
  "GET /api/v1/containers/list": {
    "success": true,
    "data": [
      { "id": 5, "uuid": "00000000-0000-4000-8000-000000000005", "name": "example-vm", "status": "running", "ip": "10.0.0.10" }
    ]
  },
  "POST /api/v1/containers/list": {
    "success": true,
    "data": [
      { "id": 5, "uuid": "00000000-0000-4000-8000-000000000005", "name": "example-vm", "status": "running", "ip": "10.0.0.10" }
    ]
  },
  "POST /api/v1/containers": {
    "success": true,
    "message": "Container created successfully"
  },
  "GET /api/v1/containers/{id|uuid|name}": {
    "success": true,
    "data": {
      "id": 5,
      "uuid": "00000000-0000-4000-8000-000000000005",
      "name": "example-vm",
      "status": "running",
      "ip": "10.0.0.10",
      "ipv6": "2001:db8:100::1005",
      "ssh_port": 22004,
      "ssh_password": "***",
      "policy_blocked": false
    }
  },
  "POST /api/v1/containers/{id}/start": {
    "success": true,
    "message": "Task queued",
    "data": { "task_id": "task-10", "container_name": "example-vm", "status": "pending", "action": "start" }
  },
  "POST /api/v1/containers/{id}/stop": {
    "success": true,
    "message": "Task queued",
    "data": { "task_id": "task-10", "container_name": "example-vm", "status": "pending", "action": "stop" }
  },
  "POST /api/v1/containers/{id}/restart": {
    "success": true,
    "message": "Task queued",
    "data": { "task_id": "task-10", "container_name": "example-vm", "status": "pending", "action": "restart" }
  },
  "POST /api/v1/containers/{id}/reinstall": {
    "success": true,
    "message": "Task queued",
    "data": { "task_id": "task-10", "container_name": "example-vm", "status": "pending", "action": "reinstall" }
  },
  "DELETE /api/v1/containers/{id}/delete": {
    "success": true,
    "message": "Task queued",
    "data": { "task_id": "task-10", "container_name": "example-vm", "status": "pending", "action": "delete" }
  },
  "GET /api/v1/containers/{id}/usage": {
    "success": true,
    "data": {
      "cpu_usage_pct": 0,
      "cpu_usage_usec": 3908852,
      "memory_usage_bytes": 29331456,
      "disk_usage_bytes": 515100672,
      "network_rx_bytes": 131232,
      "network_tx_bytes": 16828,
      "load1": 0.1,
      "load5": 0.06,
      "load15": 0.01
    }
  },
  "GET /api/v1/containers/{id}/traffic": {
    "success": true,
    "data": {
      "mode": "total",
      "limit_gb": 1024,
      "in_limit_gb": 0,
      "out_limit_gb": 0,
      "total_used_bytes": 142082,
      "rx_used_bytes": 127212,
      "tx_used_bytes": 14870,
      "used_pct": 0,
      "reset_date": "2026-06"
    }
  },
  "POST /api/v1/containers/{id}/traffic-reset": {
    "success": true,
    "message": "Traffic reset"
  },
  "PUT /api/v1/containers/{id}/traffic-limit": {
    "success": true,
    "message": "Traffic limit updated"
  },
  "PUT /api/v1/containers/{id}/resource-limit": {
    "success": true,
    "message": "Resource limits updated"
  },
  "PUT /api/v1/containers/{id}/expiry": {
    "success": true,
    "message": "Expiry updated"
  },
  "POST /api/v1/containers/{id}/reset-password": {
    "success": true,
    "message": "SSH password reset successfully",
    "data": { "password": "***" }
  },
  "POST /api/v1/containers/{id}/ipv6": {
    "success": true,
    "message": "IPv6 assigned",
    "data": { "id": 5, "name": "example-vm", "ipv6": "2001:db8:100::1005" }
  }
}
```

### Ports and Snapshots

```json
{
  "GET /api/v1/containers/{id}/random-port?host_ip=203.0.113.10": {
    "success": true,
    "data": { "port": 61320 }
  },
  "POST /api/v1/containers/{id}/port-mappings": {
    "success": true,
    "data": [
      { "container_port": 22, "host_port": 22004, "protocol": "tcp", "description": "SSH" },
      { "container_port": 8080, "host_port": 61320, "protocol": "tcp", "description": "HTTP" }
    ]
  },
  "PUT /api/v1/containers/{id}/port-mappings/{index}": {
    "success": true,
    "data": [
      { "container_port": 8081, "host_port": 61320, "protocol": "tcp", "description": "HTTP" }
    ]
  },
  "DELETE /api/v1/containers/{id}/port-mappings/{index}": {
    "success": true,
    "data": []
  },
  "GET /api/v1/containers/{id}/firewall": {
    "success": true,
    "data": {
      "enabled": true,
      "default_action": "DROP",
      "rules": [
        { "id": "a1b2c3d4", "direction": "in", "protocol": "tcp", "action": "ACCEPT", "network": "ipv4", "source_ip": "203.0.113.0/24", "port": "22,80,443", "description": "allow admin and web" }
      ]
    }
  },
  "PUT /api/v1/containers/{id}/firewall": {
    "success": true,
    "message": "Firewall updated",
    "data": { "enabled": true, "default_action": "DROP", "rules": [] }
  },
  "GET /api/v1/snapshots": {
    "success": true,
    "data": null
  },
  "GET /api/v1/containers/{id}/snapshots": {
    "success": true,
    "data": {
      "quota": 1,
      "schedule": { "enabled": false, "interval_hours": 0, "last_run": "", "next_run": "", "time": "", "created_by": "" },
      "snapshots": []
    }
  },
  "POST /api/v1/containers/{id}/snapshots": {
    "success": true,
    "data": {
      "id": "snap-20260608-001",
      "container_id": 5,
      "container_name": "example-vm",
      "created_at": "2026-06-08 16:00:00",
      "created_by": "api:Automation",
      "scheduled": false,
      "size_bytes": 10485760
    }
  },
  "DELETE /api/v1/containers/{id}/snapshots/{snapshot_id}": {
    "success": true,
    "message": "Snapshot deleted"
  },
  "POST /api/v1/containers/{id}/snapshots/{snapshot_id}/restore": {
    "success": true,
    "message": "Snapshot restored"
  },
  "POST /api/v1/containers/{id}/snapshots/schedule": {
    "success": true,
    "data": {
      "container": { "id": 5, "name": "example-vm", "snapshot_schedule_enabled": true, "snapshot_schedule_interval_hours": 24, "snapshot_schedule_time": "03:00" }
    }
  },
  "PUT /api/v1/containers/{id}/snapshots/quota": {
    "success": true,
    "data": {
      "quota": 2,
      "container": { "id": 5, "name": "example-vm", "snapshot_limit": 2 }
    }
  }
}
```

### Platform Management

```json
{
  "GET /api/v1/templates": {
    "success": true,
    "data": [
      { "id": "ubuntu-noble", "name": "Ubuntu 24.04", "distro": "ubuntu", "release": "noble", "arch": "amd64", "description": "Ubuntu 24.04 LTS" },
      { "id": "debian-bookworm", "name": "Debian 12", "distro": "debian", "release": "bookworm", "arch": "amd64", "description": "Debian 12 (Bookworm)" }
    ]
  },
  "GET /api/v1/images": {
    "success": true,
    "data": [
      { "id": "ubuntu-noble", "name": "Ubuntu 24.04", "type": "lxc", "downloaded": true, "enabled": true, "downloading": false, "progress": 0, "size_bytes": 135005452 }
    ]
  },
  "GET /api/v1/images/enabled?type=lxc": {
    "success": true,
    "data": [
      { "id": "ubuntu-noble", "name": "Ubuntu 24.04", "distro": "ubuntu", "release": "noble", "arch": "amd64", "variant": "default", "description": "Ubuntu 24.04 LTS", "type": "lxc" }
    ]
  },
  "POST /api/v1/images/download": {
    "success": true,
    "message": "Already downloaded"
  },
  "POST /api/v1/images/cancel": {
    "success": true,
    "message": "Cancel requested"
  },
  "DELETE /api/v1/images/delete": {
    "success": true,
    "message": "Deleted"
  },
  "PUT /api/v1/images/toggle": {
    "success": true,
    "message": "OK"
  },
  "GET /api/v1/security/alerts": {
    "success": true,
    "data": []
  },
  "POST /api/v1/security/check": {
    "success": true,
    "message": "Security check completed"
  },
  "GET /api/v1/security/logs?container={name}": {
    "success": true,
    "data": []
  },
  "GET /api/v1/security/summary": {
    "success": true,
    "data": { "critical": 0, "high": 0, "medium": 0, "low": 0, "total_alerts": 0 }
  },
  "GET /api/v1/security/settings": {
    "success": true,
    "data": { "auto_shutdown": false }
  },
  "PUT /api/v1/security/settings": {
    "success": true,
    "data": { "auto_shutdown": false }
  },
  "GET /api/v1/swap": {
    "success": true,
    "data": { "total_mb": 16383, "used_mb": 0, "free_mb": 16383, "enabled": true, "swap_file": "/swapfile" }
  },
  "POST /api/v1/swap": {
    "success": true,
    "message": "SWAP 已调整为 16384 MB",
    "data": { "total_mb": 16383, "used_mb": 0, "free_mb": 16383, "enabled": true, "swap_file": "/swapfile" }
  },
  "GET /api/v1/language": {
    "success": true,
    "data": { "language": "zh" }
  },
  "PUT /api/v1/language": {
    "success": true,
    "data": { "language": "en" }
  },
  "GET /api/v1/ssl": {
    "success": true,
    "data": { "enabled": true, "mode": "self-signed", "target": "panel.example.com", "detected_host": "panel.example.com", "needs_restart": false }
  },
  "PUT /api/v1/ssl": {
    "success": true,
    "message": "SSL settings saved",
    "data": { "enabled": true, "mode": "self-signed", "target": "panel.example.com", "needs_restart": true }
  },
  "GET /api/v1/webssh-origins": {
    "success": true,
    "data": { "origins": ["https://panel.example.com"], "current_origin": "https://panel.example.com" }
  },
  "PUT /api/v1/webssh-origins": {
    "success": true,
    "message": "Origin allowlist saved",
    "data": { "origins": ["https://panel.example.com"], "current_origin": "https://panel.example.com" }
  },
  "POST /api/v1/batch-create": {
    "success": true,
    "data": ["task-12"]
  },
  "POST /api/v1/batch-action": {
    "success": true,
    "data": ["task-13"]
  },
  "POST /api/v1/ssh-ticket": {
    "success": true,
    "data": { "ticket": "***60秒有效票据***" }
  },
  "POST /api/v1/vnc-ticket": {
    "success": true,
    "data": { "ticket": "***60秒有效票据***" }
  }
}
```

### Accounts and Logs

```json
{
  "POST /api/v1/sub-user/create": {
    "success": true,
    "message": "Sub-user created",
    "data": {
      "id": "sub-xxxxxxxx",
      "username": "user-xxxxxxxx",
      "password": "***",
      "container_names": ["example-vm"],
      "access_code": "********",
      "created_at": "2026-06-08 16:00:00"
    }
  },
  "GET /api/v1/sub-users": {
    "success": true,
    "data": []
  },
  "POST /api/v1/sub-users/{id}/rotate-password": {
    "success": true,
    "data": { "username": "user-xxxxxxxx", "password": "***", "access_code": "********" }
  },
  "GET /api/v1/sub-users/{id}/audit-logs": {
    "success": true,
    "data": []
  },
  "GET /api/v1/sub-users/{id}/login-logs": {
    "success": true,
    "data": []
  },
  "GET /api/v1/audit-logs": {
    "success": true,
    "data": [
      { "time": "2026-06-08 15:44:40", "action": "apikey.create", "target": "Test", "detail": "scopes=*", "user": "admin", "success": true }
    ]
  },
  "GET /api/v1/login-logs": {
    "success": true,
    "data": [
      { "time": "2026-06-08 08:24:00 UTC", "username": "admin", "ip": "198.51.100.23", "user_agent": "Mozilla/5.0 ...", "success": true }
    ]
  },
  "GET /api/v1/api-keys": {
    "success": true,
    "data": [
      { "id": "c271023f", "name": "Test", "prefix": "clicd_sk_dd9d...", "ip_whitelist": "", "created_at": "2026-06-08 15:44:40", "last_used": "2026-06-08 15:46:10", "scopes": ["*"], "expires_at": "", "disabled": false, "container_uuids": [], "last_used_ip": "198.51.100.23" }
    ]
  },
  "POST /api/v1/api-keys": {
    "success": true,
    "message": "API key created. Save this key now - it won't be shown again.",
    "data": { "id": "a1b2c3d4", "name": "Automation", "key": "clicd_sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", "prefix": "clicd_sk_xxxx...", "ip_whitelist": "198.51.100.23", "scopes": ["dashboard:read", "container:read"], "expires_at": "2026-12-31 23:59:59", "disabled": false, "container_uuids": ["00000000-0000-4000-8000-000000000005"] }
  },
  "PATCH /api/v1/api-keys/{id}": {
    "success": true,
    "data": { "id": "a1b2c3d4", "name": "Automation", "prefix": "clicd_sk_xxxx...", "scopes": ["dashboard:read", "container:read"], "expires_at": "2026-12-31 23:59:59", "disabled": false, "container_uuids": ["00000000-0000-4000-8000-000000000005"] }
  },
  "DELETE /api/v1/api-keys/{id}": {
    "success": true,
    "message": "API key deleted"
  }
}
```
