import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bookmark } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate } from "react-router-dom";
import type { JobSummary } from "../../api/models/job";
import type { SavedJob } from "../../api/models/candidate";
import { repositories } from "../../api/repositories";
import { useAuth } from "../../app/authContext";
import { useToast } from "../ui/useToast";

export function JobSaveButton({
  job,
  compact = false,
}: {
  job: JobSummary;
  compact?: boolean;
}) {
  const { t } = useTranslation();
  const auth = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const client = useQueryClient();
  const toast = useToast();
  const candidate = auth.session?.user.role === "job_seeker";
  const saved = useQuery({
    queryKey: ["saved-jobs"],
    queryFn: () => repositories.candidate.listSavedJobs(),
    enabled: candidate,
  });
  const isSaved = saved.data?.some((item) => item.jobId === job.id) ?? false;
  const mutation = useMutation({
    mutationFn: (saving: boolean) =>
      saving
        ? repositories.candidate.saveJob(job.id, auth.session!.csrfToken)
        : repositories.candidate.unsaveJob(job.id, auth.session!.csrfToken),
    onMutate: async (saving) => {
      await client.cancelQueries({ queryKey: ["saved-jobs"] });
      const previous = client.getQueryData<SavedJob[]>(["saved-jobs"]);
      client.setQueryData<SavedJob[]>(["saved-jobs"], (current = []) =>
        saving
          ? [
              {
                jobId: job.id,
                savedAt: new Date().toISOString(),
                available: true,
                title: job.title,
                companyName: job.company.name,
                location: job.location,
                source: job.source.id,
                sourceName: job.source.name,
                applicationMethod: job.application.method,
              },
              ...current,
            ]
          : current.filter((item) => item.jobId !== job.id),
      );
      return { previous };
    },
    onError: (_error, _variables, context) => {
      client.setQueryData(["saved-jobs"], context?.previous);
      toast.showToast({ tone: "error", title: t("saved.error") });
    },
    onSuccess: (_data, saving) =>
      toast.showToast({ title: t(saving ? "saved.added" : "saved.removed") }),
    onSettled: () =>
      void client.invalidateQueries({ queryKey: ["saved-jobs"] }),
  });
  if (auth.session && !candidate) return null;
  const activate = () => {
    if (!auth.session) {
      const returnTo = `${location.pathname}${location.search}`;
      navigate(`/login?returnTo=${encodeURIComponent(returnTo)}`);
      return;
    }
    mutation.mutate(!isSaved);
  };
  return (
    <button
      type="button"
      className={`job-save-button ${isSaved ? "is-saved" : ""} ${compact ? "is-compact" : ""}`}
      aria-pressed={isSaved}
      aria-label={t(isSaved ? "saved.unsave" : "saved.save")}
      onClick={activate}
      disabled={mutation.isPending}
    >
      <Bookmark
        size={18}
        fill={isSaved ? "currentColor" : "none"}
        aria-hidden="true"
      />
      {compact ? null : (
        <span>{t(isSaved ? "saved.saved" : "saved.save")}</span>
      )}
    </button>
  );
}
