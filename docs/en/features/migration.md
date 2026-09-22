# Node Migration

Node migration moves container configuration between servers. It works by "exporting a migration bundle → importing it on the target node". The bundle contains the container configuration (network, port mappings, SSH info, etc.); after import, the container can be managed on the target node.

## Export a Migration Bundle

1. Go to the "Node Migration" page.
2. Select the container(s) to migrate in the container list and click "Export".
3. The browser downloads a `<container-name>.migrate.json` file.

The export is a snapshot of the container configuration and does not include the rootfs data. To migrate data, create a snapshot on the source node or back up the container data directory yourself, then restore it on the target node.

## Import a Migration Bundle

1. On the target node's "Node Migration" page, select the exported `.migrate.json` file.
2. The page parses the bundle and shows the container info.
3. Confirm and click "Import"; the target node rebuilds the container record from the configuration.

After importing, verify that the container status, network, and port mappings match the source node.

## Use Cases

- Moving to a new host or migrating to a server with more resources.
- Moving a container from one worker node to another under the same Controller.
- Disaster recovery drills: export configurations periodically and restore quickly when needed.

## Related Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/migrate/export/{id}` | Export a container migration bundle |
| POST | `/api/migrate/import` | Import a migration bundle |

`/api/v1/migrate/import` also provides a versioned import endpoint.
