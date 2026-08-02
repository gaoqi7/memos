import { CheckCircle2Icon, ChevronLeft, ChevronRight, LoaderIcon, RefreshCwIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { handleError } from "@/lib/error";
import { listImmichAssets, type ImmichAsset } from "../services/immichService";

interface ImmichPickerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  attachedAssetIds: string[];
  onApplySelection: (selectedAssetIds: string[]) => Promise<void>;
}

const PAGE_SIZE = 35;

const ImmichPickerDialog = ({ open, onOpenChange, attachedAssetIds, onApplySelection }: ImmichPickerDialogProps) => {
  const [assets, setAssets] = useState<ImmichAsset[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [nextPageToken, setNextPageToken] = useState<string | undefined>();
  const [isLoading, setIsLoading] = useState(false);
  const [isSelecting, setIsSelecting] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const hasNextPage = useMemo(() => Boolean(nextPageToken), [nextPageToken]);
  const hasPrevPage = currentPage > 1;
  const attachedIdSet = useMemo(() => new Set(attachedAssetIds), [attachedAssetIds]);
  const hasSelectionChanged = useMemo(() => {
    if (selectedIds.size !== attachedIdSet.size) {
      return true;
    }
    for (const id of selectedIds) {
      if (!attachedIdSet.has(id)) {
        return true;
      }
    }
    return false;
  }, [attachedIdSet, selectedIds]);

  const fetchPage = useCallback(
    async (page: number) => {
      const pageToken = page > 1 ? String(page) : undefined;
      const response = await listImmichAssets({
        pageSize: PAGE_SIZE,
        pageToken,
      });

      setAssets(response.assets);
      setNextPageToken(response.nextPageToken || undefined);
    },
    [],
  );

  useEffect(() => {
    if (!open) {
      setSelectedIds(new Set());
      setCurrentPage(1);
      return;
    }
    setSelectedIds(new Set(attachedAssetIds));
    setCurrentPage(1);
    setIsLoading(true);
    fetchPage(1)
      .catch((error) => {
        handleError(error, toast.error, {
          context: "Failed to fetch Immich assets",
          fallbackMessage: "Failed to load Immich assets.",
        });
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [attachedAssetIds, fetchPage, open]);

  const handleRefresh = useCallback(async () => {
    setIsLoading(true);
    try {
      await fetchPage(currentPage);
    } catch (error) {
      handleError(error, toast.error, {
        context: "Failed to refresh Immich assets",
        fallbackMessage: "Failed to load Immich assets.",
      });
    } finally {
      setIsLoading(false);
    }
  }, [currentPage, fetchPage]);

  const handleGoToPage = useCallback(
    async (page: number) => {
      if (page < 1) return;
      setIsLoading(true);
      try {
        await fetchPage(page);
        setCurrentPage(page);
      } catch (error) {
        handleError(error, toast.error, {
          context: `Failed to fetch page ${page} of Immich assets`,
          fallbackMessage: "Failed to load Immich assets.",
        });
      } finally {
        setIsLoading(false);
      }
    },
    [fetchPage],
  );

  const handlePrevPage = useCallback(async () => {
    if (!hasPrevPage) return;
    await handleGoToPage(currentPage - 1);
  }, [currentPage, handleGoToPage, hasPrevPage]);

  const handleNextPage = useCallback(async () => {
    if (!hasNextPage) return;
    const nextPage = Number(nextPageToken);
    await handleGoToPage(isNaN(nextPage) ? currentPage + 1 : nextPage);
  }, [currentPage, handleGoToPage, hasNextPage, nextPageToken]);

  const handleSelectAsset = useCallback(
    (asset: ImmichAsset) => {
      if (isSelecting) {
        return;
      }
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (next.has(asset.id)) {
          next.delete(asset.id);
        } else {
          next.add(asset.id);
        }
        return next;
      });
    },
    [isSelecting],
  );

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (nextOpen) {
        onOpenChange(true);
        return;
      }
      if (!hasSelectionChanged || isSelecting) {
        onOpenChange(false);
        return;
      }
      setIsSelecting(true);
      onApplySelection(Array.from(selectedIds))
        .catch((error) => {
          handleError(error, toast.error, {
            context: "Failed to attach Immich assets",
            fallbackMessage: "Failed to attach Immich assets.",
          });
        })
        .finally(() => {
          setIsSelecting(false);
          onOpenChange(false);
        });
    },
    [hasSelectionChanged, isSelecting, onApplySelection, onOpenChange, selectedIds],
  );

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden flex flex-col gap-4">
        <DialogHeader>
          <DialogTitle>Immich</DialogTitle>
          <DialogDescription>Select one or more photos/videos, then close to attach.</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-2">
            <div className="text-sm text-muted-foreground">All photos</div>
            <Button variant="outline" size="icon" onClick={handleRefresh} disabled={isLoading}>
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

            <div className="grid grid-cols-5 gap-3">
              {assets.map((asset) => (
                <button
                  key={asset.id}
                  type="button"
                  className={`group relative rounded-md overflow-hidden border bg-background hover:border-primary transition ${
                    selectedIds.has(asset.id) ? "border-primary ring-2 ring-primary/40" : ""
                  }`}
                  onClick={() => handleSelectAsset(asset)}
                >
                  <img
                    src={asset.thumbnailUrl}
                    alt={asset.filename}
                    className="h-28 w-full object-cover"
                    loading="lazy"
                  />
                  <div className="absolute inset-0 opacity-0 group-hover:opacity-100 bg-black/40 transition" />
                  {selectedIds.has(asset.id) ? (
                    <div className="absolute top-2 right-2 text-primary bg-background/90 rounded-full">
                      <CheckCircle2Icon className="size-5" />
                    </div>
                  ) : null}
                  <div className="absolute bottom-0 left-0 right-0 text-xs text-white bg-black/50 px-2 py-1 truncate">
                    {asset.filename}
                  </div>
                </button>
              ))}
            </div>

            <div className="flex items-center justify-center gap-2 mt-4">
              <Button
                variant="outline"
                size="sm"
                onClick={handlePrevPage}
                disabled={!hasPrevPage || isLoading}
              >
                <ChevronLeft className="size-4" />
                Previous
              </Button>
              <span className="text-sm text-muted-foreground px-3">Page {currentPage}</span>
              <Button
                variant="outline"
                size="sm"
                onClick={handleNextPage}
                disabled={!hasNextPage || isLoading}
              >
                Next
                <ChevronRight className="size-4" />
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ImmichPickerDialog;
