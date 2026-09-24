import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FileText, Send } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation, useNavigate } from "react-router-dom";
import type { JobSummary } from "../../api/models/job";
import { repositories } from "../../api/repositories";
import { useAuth } from "../../app/authContext";
import { Button } from "../ui/Button";
import { Textarea } from "../ui/FormControls";
import { Modal } from "../ui/Overlays";
import { useToast } from "../ui/useToast";

export function ApplyAction({ job }: { job: JobSummary }) {
  const { t } = useTranslation();
  const auth = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const client = useQueryClient();
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [message, setMessage] = useState("");
  const candidate = auth.session?.user.role === "job_seeker";
  const profile = useQuery({
    queryKey: ["candidate-profile"],
    queryFn: () => repositories.candidate.getProfile(),
    enabled: candidate && open,
  });
  const applications = useQuery({
    queryKey: ["applications"],
    queryFn: () => repositories.candidate.listApplications(),
    enabled: candidate,
  });
  const existing = applications.data?.find((item) => item.jobId === job.id);
  const apply = useMutation({
    mutationFn: () =>
      repositories.candidate.apply(
        job.id,
        profile.data!.resume!.id,
        message,
        auth.session!.csrfToken,
      ),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["applications"] });
      setOpen(false);
      toast.showToast({ title: t("applications.submitted") });
    },
    onError: () =>
      toast.showToast({ tone: "error", title: t("applications.submitError") }),
  });
  if (job.source.type !== "native" || job.application.method !== "internal")
    return null;
  if (auth.session && !candidate) return null;
  if (existing)
    return (
      <Link
        className="ui-button ui-button--secondary ui-button--md"
        to="/applications"
      >
        {t("applications.alreadyApplied")}
      </Link>
    );
  const begin = () => {
    if (!auth.session) {
      navigate(
        `/login?returnTo=${encodeURIComponent(`${location.pathname}${location.search}`)}`,
      );
      return;
    }
    setOpen(true);
  };
  return (
    <>
      <Button leadingIcon={<Send size={18} />} onClick={begin}>
        {t("applications.apply")}
      </Button>
      <Modal
        open={open}
        onClose={() => setOpen(false)}
        title={t("applications.applyTitle")}
        description={t("applications.applyDescription")}
        closeLabel={t("common.close")}
        footer={
          profile.data?.resume ? (
            <>
              <Button variant="secondary" onClick={() => setOpen(false)}>
                {t("common.cancel")}
              </Button>
              <Button
                isLoading={apply.isPending}
                onClick={() => apply.mutate()}
              >
                {t("applications.submit")}
              </Button>
            </>
          ) : undefined
        }
      >
        {profile.isPending ? (
          <div
            className="application-dialog-skeleton"
            aria-label={t("applications.loadingResume")}
          />
        ) : null}
        {profile.isError ? (
          <p className="auth-error" role="alert">
            {t("applications.profileError")}
          </p>
        ) : null}
        {profile.data && !profile.data.resume ? (
          <div className="application-resume-empty">
            <FileText size={24} />
            <strong>{t("applications.resumeRequired")}</strong>
            <p>{t("applications.resumeRequiredHint")}</p>
            <Link
              className="ui-button ui-button--primary ui-button--md"
              to="/profile"
            >
              {t("applications.goToProfile")}
            </Link>
          </div>
        ) : null}
        {profile.data?.resume ? (
          <>
            <div className="selected-resume">
              <FileText size={22} />
              <div>
                <strong>{profile.data.resume.originalFilename}</strong>
                <span>{formatBytes(profile.data.resume.sizeBytes)}</span>
              </div>
            </div>
            <Textarea
              label={t("applications.message")}
              value={message}
              maxLength={3000}
              rows={5}
              onChange={(event) => setMessage(event.target.value)}
              hint={t("applications.messageHint")}
            />
          </>
        ) : null}
      </Modal>
    </>
  );
}
function formatBytes(value: number) {
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}
