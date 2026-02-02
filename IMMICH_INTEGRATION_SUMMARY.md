# Immich Integration Summary (Project Handoff)

## Goal
Add an Immich photo picker to Memos so users can browse Immich photos from the memo editor, select one, and attach it as a real image (not just a link). Attachments remain external (no duplicate storage). When a photo is attached, it should be added to a dedicated Immich album (default "Memos"). Configuration is instance-wide via environment variables.

## Features Added
- Immich picker button in the memo "+" menu (now first item).
- Immich asset browsing (All photos only; no album selection in UI).
- Attachment created as `immich://<asset-id>` and displayed as image via proxy.
- Optional auto-add attached asset to an Immich album (default "Memos").
- Immich proxy for thumbnails and previews.

## Environment Variables
```
MEMOS_IMMICH_URL=https://img.gaoqi7.com
MEMOS_IMMICH_API_KEY=<key>
MEMOS_IMMICH_ALBUM_NAME=Memos   # optional (default "Memos")
MEMOS_IMMICH_ALBUM_ID=<uuid>    # optional override if set
```

Important: Memos does **not** auto-load `.env`. Use:
```
set -a; source .env; set +a
```

## Backend Changes
- `internal/immich/immich.go`
  - Immich client, config, album add, asset list/search.
  - Robust decoding for multiple Immich response shapes.
  - Asset ID normalization (strip leading `/`).
  - Album add via `POST/PUT /api/albums/{id}/assets`.
- `server/router/immich/immich.go`
  - JSON endpoint for frontend picker: `GET /api/immich/assets`.
- `server/router/fileserver/fileserver.go`
  - Proxy route: `GET /file/immich/:assetID` (thumbnail/fullsize/original).
- `server/router/api/v1/attachment_service.go`
  - Attachment creation tolerates asset-info 404s (logs warn; still attaches).
- `server/server.go`
  - Registers Immich router.

## Frontend Changes
- `web/src/components/MemoEditor/Toolbar/InsertMenu.tsx`
  - Immich button is first item.
- `web/src/components/MemoEditor/components/ImmichPickerDialog.tsx`
  - Picker dialog (All photos only).
- `web/src/components/MemoEditor/services/immichService.ts`
  - Fetch Immich assets.
- `web/src/components/MemoEditor/components/index.ts`
  - Exports picker dialog.

## Album Auto-Add
- Adds attached asset to Immich album using:
  - `POST /api/albums/{id}/assets` with `{ "ids": [assetID] }`
  - Tries PUT then POST for compatibility.
- Skips add if asset ID is not a valid UUID.

## Fixes Applied
- Asset ID sometimes returned as `/uuid` or in other fields (`assetId`, `deviceAssetId`, `uuid`). Decoder now handles all and strips leading `/`.
- Do not fail attachment creation if Immich asset info endpoint 404s.
- Immich picker Select error fixed (no empty SelectItem values).
- Album selection removed per user request (All photos only).

## Testing
- `go test ./...` runs; `store/test` fails without Docker/Testcontainers.
- Frontend build requires Node + pnpm.

## Git / Commits
- Fork remote: `https://github.com/gaoqi7/memos.git`
- Commits:
  - `5d591667` Add Immich picker integration
  - `460e0e65` Move Immich insert to top

## Frontend Build
```
cd /home/rick/Documents/project/memos/web
pnpm install
pnpm release
```
Then commit/push `server/router/frontend/dist/*`.

## Docker Usage
```
docker build -t memos-immich:latest .
docker run -d --name memos -p 8081:8081 \
  -v ~/.memos:/var/opt/memos \
  -e MEMOS_IMMICH_URL="https://img.gaoqi7.com" \
  -e MEMOS_IMMICH_API_KEY="..." \
  memos-immich:latest
```

## Known Issues
- Some Immich versions differ in asset/info endpoints; code uses multiple fallbacks.
- Asset info 404s are tolerated (attachment still created).
