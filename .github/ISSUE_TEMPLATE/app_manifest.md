---
name: 🧩 New App Manifest
about: Submit a new app for the Sovereign Stack marketplace
title: "[App] "
labels: ["app-manifest", "community"]
---

## App Details

**Name:** <!-- e.g., Jellyfin -->
**Docker Image:** <!-- e.g., jellyfin/jellyfin:10.9 -->
**Category:** <!-- productivity / media / monitoring / development / system / ai / security / network / automation / smart-home / analytics / lifestyle / finance -->
**Website:** <!-- e.g., https://jellyfin.org -->
**Version:** <!-- e.g., 10.9 -->

## Docker Compose Snippet

```yaml
services:
  app-name:
    image: your/image:tag
    ports:
      - "HOST:CONTAINER"
    volumes:
      - app_data:/app/data
    environment:
      - ENV_VAR=value
```

## Requirements

- [ ] Requires PostgreSQL
- [ ] Requires Redis
- Minimum RAM: <!-- e.g., 512 MB -->
- Minimum Disk: <!-- e.g., 5 GB -->

## Caddy Route (optional)

```
Path: /app-name
Port: HOST_PORT
```

## Testing

- [ ] I have tested this app with `docker compose up`
- [ ] The app starts and is accessible
- [ ] The Caddy route works correctly

## Additional Notes

<!-- Any special configuration, known issues, or tips -->
