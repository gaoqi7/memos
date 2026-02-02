import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useTranslate } from "@/utils/i18n";

interface ExternalLinkDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (link: string) => void | Promise<void>;
  isSubmitting?: boolean;
}

const ExternalLinkDialog = ({ open, onOpenChange, onConfirm, isSubmitting }: ExternalLinkDialogProps) => {
  const t = useTranslate();
  const [link, setLink] = useState("");

  useEffect(() => {
    if (!open) {
      setLink("");
    }
  }, [open]);

  const trimmedLink = link.trim();
  const handleConfirm = async () => {
    if (!trimmedLink) return;
    await onConfirm(trimmedLink);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("resource.create-dialog.external-link.option")}</DialogTitle>
        </DialogHeader>
        <div className="w-full flex flex-col gap-2">
          <label className="text-sm font-medium">{t("resource.create-dialog.external-link.link")}</label>
          <Input
            autoFocus
            value={link}
            onChange={(event) => setLink(event.target.value)}
            placeholder={t("resource.create-dialog.external-link.link-placeholder")}
            disabled={isSubmitting}
          />
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleConfirm} disabled={!trimmedLink || isSubmitting}>
            {t("common.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ExternalLinkDialog;
