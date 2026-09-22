# Introduction

EyvesCloud is a lightweight virtualization management panel for LXC/KVM (deeply reworked from CLICD). It consolidates common host operations into a web console and CLI, and adds "Controller-Agent" multi-node management, making it suitable for VPS providers, labs, developers running their own virtualization nodes, and scenarios where container access needs to be distributed in batches.

## Core Capabilities

- Manage LXC containers and KVM virtual machines: create, start, stop, restart, reinstall, delete, reset passwords, and batch operations.
- Controller-Agent multi-node: the Controller generates a one-line install script; after the worker server runs it, it registers automatically, and the Controller can view/operate its containers directly.
- Configure CPU, memory, disk, Swap, independent download/upload bandwidth, read/write I/O limits, traffic limits, and expiration time, with automatic shutdown on overage.
- Manage NAT4 port mappings and assign public IPv6 when the host has IPv6 routing.
- Open WebSSH or WebVNC from the browser.
- Manage image downloads, enablement status, and local cache.
- Create, restore, and delete snapshots, plus scheduled snapshots and snapshot quotas.
- Storage pool management, resource quotas, and a policy engine, with tenant isolation and per-container authorization.
- Generate security alerts based on connection behavior, and keep audit logs and login logs.
- Create sub-user access links for specific containers, with view/operate scope limits.
- Integrate automation through API keys and the `/api/v1` interface.

## Use Cases

- Quickly allocate multiple Linux containers on a single host.
- Unify multiple standalone servers under one Controller panel (like the node mode of MagicCloud).
- Grant users temporary access to container consoles, SSH, VNC, or NAT port management.
- Automate container creation, resource adjustments, password resets, or resource reclamation through the API.
- Need a panel that is more intuitive than pure CLI but not a heavyweight platform.

## Tech Stack

- Backend: Go, `net/http`, SQLite, systemd, LXC, KVM/libvirt, cgroup v2, iptables, conntrack.
- Frontend: React, TypeScript, Vite, Tailwind CSS, lucide-react, xterm.js, noVNC.
- Release: GitHub Actions builds Linux AMD64/ARM64 release artifacts; the install script fetches the latest Release by default.
