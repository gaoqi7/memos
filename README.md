# Memos

An open-source, self-hosted note-taking service. This fork adds deep [Immich](https://immich.app/) integration — combine text, photos, and videos in your journal.

## Immich Integration

This fork connects Memos to your self-hosted Immich instance, letting you attach photos and videos from your Immich library directly into your notes.

### Setup

Set these environment variables when running the container:

```bash
MEMOS_IMMICH_URL=https://your-immich-instance.com
MEMOS_IMMICH_API_KEY=your-immich-api-key
```

Optional — auto-add attached assets to a specific Immich album:

```bash
MEMOS_IMMICH_ALBUM_NAME=Memos       # default album name
MEMOS_IMMICH_ALBUM_ID=album-uuid    # or use an existing album ID
```

### How It Works

1. **Browse & attach** — Click the Immich button in the editor toolbar to open the picker. Browse all your Immich photos and videos with pagination (5-column grid, 35 per page). Select one or multiple items.

2. **No duplicate storage** — Attachments reference Immich assets via external links. Photos and videos are proxied through Memos but stored only in Immich. No wasted disk space.

3. **Auto album** — Attached assets are optionally added to a dedicated Immich album (default: "Memos") so you can easily find everything you've referenced in your notes.

4. **Video playback** — Video attachments play inline in notes with full controls. Click a video thumbnail to open a fullscreen lightbox with autoplay. Videos from Immich stream with proper range-request support.

5. **Image lightbox** — Click any image to open a fullscreen preview dialog.

### Immich Picker

- **Page navigation** — Previous/Next buttons to browse your entire Immich library
- **5-column grid** — 35 thumbnails per page for quick scanning
- **Multi-select** — Pick multiple photos and videos at once
- **Selection tracking** — Already-attached assets show as selected when you reopen the picker

### Architecture

```
Memos Editor → Immich Picker Dialog → GET /api/immich/assets → Immich API /api/search/metadata
                                                                  (falls back to /api/assets)
Memos Viewer → <video>/<img> → GET /file/attachments/:uid/:filename → Immich API proxy
```

Attachments are stored with `immich:{assetId}` references and `EXTERNAL` storage type. The file server proxies requests to Immich, forwarding Range headers for video streaming.

## Quick Start

### Docker

```bash
docker run -d \
  --name memos \
  -p 5230:5230 \
  -v ~/.memos:/var/opt/memos \
  -e MEMOS_IMMICH_URL=https://your-immich.com \
  -e MEMOS_IMMICH_API_KEY=your-api-key \
  neosmemo/memos:stable
```

Open `http://localhost:5230` and start writing.

### Docker Compose

```yaml
services:
  memos:
    image: neosmemo/memos:stable
    container_name: memos
    ports:
      - "5230:5230"
    volumes:
      - ~/.memos:/var/opt/memos
    environment:
      - MEMOS_IMMICH_URL=https://your-immich.com
      - MEMOS_IMMICH_API_KEY=your-api-key
      - MEMOS_IMMICH_ALBUM_NAME=Memos
    restart: unless-stopped
```

## Features

- **Privacy-first** — Self-hosted, zero telemetry, full data ownership
- **Markdown native** — Full markdown support with plain text storage
- **Immich integration** — Browse and attach photos/videos from your Immich library
- **Video support** — Inline playback with fullscreen lightbox and autoplay
- **Image lightbox** — Fullscreen preview for attached images
- **Blazing fast** — Go backend + React frontend, optimized for performance
- **Simple deployment** — One-line Docker, supports SQLite/MySQL/PostgreSQL
- **Developer-friendly** — Full REST and gRPC APIs
- **Beautiful interface** — Clean design, dark mode, mobile-responsive

## License

Memos is open-source software licensed under the [MIT License](LICENSE).
