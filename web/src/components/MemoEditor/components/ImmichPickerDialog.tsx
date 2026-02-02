import { LoaderIcon, RefreshCwIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { handleError } from "@/lib/error";
import { listImmichAssets, type ImmichAsset } from "../services/immichService";

interface ImmichPickerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelectAsset: (asset: ImmichAsset) => Promise<void>;
}

const PAGE_SIZE = 60;

const ImmichPickerDialog = ({ open, onOpenChange, onSelectAsset }: ImmichPickerDialogProps) => {
  const [assets, setAssets] = useState<ImmichAsset[]>([]);
  const [nextPageToken, setNextPageToken] = useState<string | undefined>();
  const [isLoading, setIsLoading] = useState(false);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [isSelecting, setIsSelecting] = useState(false);

  const hasMore = useMemo(() => Boolean(nextPageToken), [nextPageToken]);

  const fetchAssets = useCallback(
    async (options: { reset?: boolean; pageToken?: string } = {}) => {
      const response = await listImmichAssets({
        pageSize: PAGE_SIZE,
        pageToken: options.pageToken,
      });

      setAssets((prev) => (options.reset ? response.assets : [...prev, ...response.assets]));
      setNextPageToken(response.nextPageToken || "");
    },
    [],
  );

  useEffect(() => {
    if (!open) {
      return;
    }
    setIsLoading(true);
    fetchAssets({ reset: true, pageToken: undefined })
      .catch((error) => {
        handleError(error, toast.error, {
          context: "Failed to fetch Immich assets",
          fallbackMessage: "Failed to load Immich assets.",
        });
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [fetchAssets, open]);

  const handleLoadMore = useCallback(async () => {
    if (!nextPageToken) {
      return;
    }
    setIsLoadingMore(true);
    try {
      await fetchAssets({ pageToken: nextPageToken });
    } catch (error) {
      handleError(error, toast.error, {
        context: "Failed to load more Immich assets",
        fallbackMessage: "Failed to load more Immich assets.",
      });
    } finally {
      setIsLoadingMore(false);
    }
  }, [fetchAssets, nextPageToken]);

  const handleSelectAsset = useCallback(
    async (asset: ImmichAsset) => {
      if (isSelecting) {
        return;
      }
      setIsSelecting(true);
      try {
        await onSelectAsset(asset);
        onOpenChange(false);
      } catch (error) {
        handleError(error, toast.error, {
          context: "Failed to attach Immich asset",
          fallbackMessage: "Failed to attach Immich asset.",
        });
      } finally {
        setIsSelecting(false);
      }
    },
    [isSelecting, onOpenChange, onSelectAsset],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden flex flex-col gap-4">
        <DialogHeader>
          <DialogTitle>Immich</DialogTitle>
          <DialogDescription>Select a photo from your Immich library to attach.</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-2">
            <div className="text-sm text-muted-foreground">All photos</div>
            <Button variant="outline" size="icon" onClick={() => fetchAssets({ reset: true })} disabled={isLoading}>
              {isLoading ? <LoaderIcon className="size-4 animate-spin" /> : <RefreshCwIcon className="size-4" />}
            </Button>
          </div>

          <div className="flex-1 overflow-y-auto border rounded-md p-3">
            {isLoading && assets.length === 0 ? (
              <div className="flex items-center justify-center py-16 text-muted-foreground">
                <LoaderIcon className="size-5 animate-spin mr-2" />
                Loading Immich assets...
              </div>
            ) : null}

            {!isLoading && assets.length === 0 ? (
              <div className="flex items-center justify-center py-16 text-muted-foreground">No assets found.</div>
            ) : null}

            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
              {assets.map((asset) => (
                <button
                  key={asset.id}
                  type="button"
                  className="group relative rounded-md overflow-hidden border bg-background hover:border-primary transition"
                  onClick={() => handleSelectAsset(asset)}
                >
                  <img
                    src={asset.thumbnailUrl}
                    alt={asset.filename}
                    className="h-32 w-full object-cover"
                    loading="lazy"
                  />
                  <div className="absolute inset-0 opacity-0 group-hover:opacity-100 bg-black/40 transition" />
                  <div className="absolute bottom-0 left-0 right-0 text-xs text-white bg-black/50 px-2 py-1 truncate">
                    {asset.filename}
                  </div>
                </button>
              ))}
            </div>

            {hasMore ? (
              <div className="flex justify-center mt-4">
                <Button variant="outline" onClick={handleLoadMore} disabled={isLoadingMore}>
                  {isLoadingMore ? <LoaderIcon className="size-4 animate-spin mr-2" /> : null}
                  Load more
                </Button>
              </div>
            ) : null}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ImmichPickerDialog;
