import { LoaderIcon, SparklesIcon } from "lucide-react";
import { type FC, useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { handleError } from "@/lib/error";
import { assistWriting, type WritingAssistMode } from "../services/writingAssistantService";
import { useEditorContext } from "../state";

const modeLabelMap: Record<WritingAssistMode, string> = {
  grammar: "Fix grammar",
  rewrite: "Rewrite naturally",
  explain: "Explain mistakes",
};

export const WritingAssistant: FC = () => {
  const { state, dispatch, actions } = useEditorContext();
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState("");
  const [resultMode, setResultMode] = useState<WritingAssistMode | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
    };
  }, []);

  const handleAssist = async (mode: WritingAssistMode) => {
    const content = state.content.trim();
    if (!content) {
      toast.error("Write some content first.");
      return;
    }

    abortControllerRef.current?.abort();
    const controller = new AbortController();
    abortControllerRef.current = controller;

    setIsOpen(true);
    setIsLoading(true);
    setResult("");
    setResultMode(mode);

    try {
      const response = await assistWriting({ content, mode }, controller.signal);
      setResult(response.output);
      setResultMode(response.mode);
    } catch (error) {
      if (controller.signal.aborted) {
        return;
      }
      handleError(error, toast.error, {
        context: "Writing assistant",
        fallbackMessage: "Failed to get writing suggestion.",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleApply = () => {
    if (!result.trim()) {
      return;
    }
    dispatch(actions.updateContent(result));
    toast.success("Applied suggestion to memo.");
  };

  return (
    <div className="w-full flex flex-col gap-2 border border-border rounded-md px-3 py-2">
      <div className="w-full flex items-center justify-between gap-2">
        <div className="flex items-center gap-1 text-sm text-muted-foreground">
          <SparklesIcon className="size-4" />
          Writing Assistant
        </div>
        <Button variant="ghost" size="sm" onClick={() => setIsOpen((v) => !v)}>
          {isOpen ? "Hide" : "Show"}
        </Button>
      </div>

      {isOpen && (
        <div className="w-full flex flex-col gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => handleAssist("grammar")} disabled={isLoading}>
              {isLoading && resultMode === "grammar" ? <LoaderIcon className="size-4 animate-spin" /> : null}
              {modeLabelMap.grammar}
            </Button>
            <Button variant="outline" size="sm" onClick={() => handleAssist("rewrite")} disabled={isLoading}>
              {isLoading && resultMode === "rewrite" ? <LoaderIcon className="size-4 animate-spin" /> : null}
              {modeLabelMap.rewrite}
            </Button>
            <Button variant="outline" size="sm" onClick={() => handleAssist("explain")} disabled={isLoading}>
              {isLoading && resultMode === "explain" ? <LoaderIcon className="size-4 animate-spin" /> : null}
              {modeLabelMap.explain}
            </Button>
          </div>

          <Textarea
            readOnly
            value={result}
            placeholder={isLoading ? "Generating suggestion..." : "Suggestions will appear here."}
            className="min-h-28 resize-y"
          />

          {resultMode !== "explain" && (
            <div className="flex justify-end">
              <Button size="sm" onClick={handleApply} disabled={!result.trim() || isLoading}>
                Apply to memo
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
