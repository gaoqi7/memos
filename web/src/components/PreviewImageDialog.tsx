import { X } from "lucide-react";
import React, { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  imgUrls: string[];
  initialIndex?: number;
  mediaType?: string;
}

function PreviewImageDialog({ open, onOpenChange, imgUrls, initialIndex = 0, mediaType }: Props) {
  const [currentIndex, setCurrentIndex] = useState(initialIndex);

  const isVideo = mediaType?.startsWith("video/");

  // Update current index when initialIndex prop changes
  useEffect(() => {
    setCurrentIndex(initialIndex);
  }, [initialIndex]);

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (!open) return;

      switch (event.key) {
        case "Escape":
          onOpenChange(false);
          break;
        default:
          break;
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, onOpenChange]);

  const handleClose = () => {
    onOpenChange(false);
  };

  const handleBackdropClick = (event: React.MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      handleClose();
    }
  };

  // Return early if no media provided
  if (!imgUrls.length) return null;

  // Ensure currentIndex is within bounds
  const safeIndex = Math.max(0, Math.min(currentIndex, imgUrls.length - 1));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="!w-[100vw] !h-[100vh] !max-w-[100vw] !max-h-[100vw] p-0 border-0 shadow-none bg-transparent [&>button]:hidden"
        aria-describedby="image-preview-description"
      >
        {/* Close button */}
        <div className="fixed top-4 right-4 z-50">
          <Button
            onClick={handleClose}
            variant="secondary"
            size="icon"
            className="rounded-full bg-popover/20 hover:bg-popover/30 border-border/20 backdrop-blur-sm"
            aria-label="Close preview"
          >
            <X className="h-4 w-4 text-popover-foreground" />
          </Button>
        </div>

        {/* Media container */}
        <div className="w-full h-full flex items-center justify-center p-4 sm:p-8 overflow-auto" onClick={handleBackdropClick}>
          {isVideo ? (
            <video
              src={imgUrls[safeIndex]}
              controls
              playsInline
              autoPlay
              className="max-w-full max-h-full object-contain select-none"
            />
          ) : (
            <img
              src={imgUrls[safeIndex]}
              alt={`Preview image ${safeIndex + 1} of ${imgUrls.length}`}
              className="max-w-full max-h-full object-contain select-none"
              draggable={false}
              loading="eager"
              decoding="async"
            />
          )}
        </div>

        {/* Screen reader description */}
        <div id="image-preview-description" className="sr-only">
          Media preview dialog. Press Escape to close or click outside.
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default PreviewImageDialog;
