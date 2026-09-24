import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, FileText, Trash2, Upload } from "lucide-react";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { CandidateWorkspaceLayout } from "../components/candidate/CandidateWorkspaceLayout";
import { Button } from "../components/ui/Button";
import { ErrorState, Skeleton } from "../components/ui/Feedback";
import { useToast } from "../components/ui/useToast";

export function ResumePage() {
  const { t, i18n } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const fileRef = useRef<HTMLInputElement>(null);
  const profile = useQuery({
    queryKey: ["candidate-profile"],
    queryFn: () => repositories.candidate.getProfile(),
  });
  const upload = useMutation({
    mutationFn: (file: File) =>
      repositories.candidate.uploadResume(file, session!.csrfToken),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["candidate-profile"] });
      toast.showToast({ title: t("resume.uploaded") });
    },
    onError: () =>
      toast.showToast({ tone: "error", title: t("resume.uploadError") }),
  });
  const remove = useMutation({
    mutationFn: () => repositories.candidate.deleteResume(session!.csrfToken),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["candidate-profile"] });
      toast.showToast({ title: t("resume.deleted") });
    },
    onError: () =>
      toast.showToast({ tone: "error", title: t("resume.deleteError") }),
  });

  const chooseFile = (files: FileList | null) => {
    const file = files?.[0];
    if (!file) return;
    if (file.type !== "application/pdf" || file.size > 10 * 1024 * 1024) {
      toast.showToast({
        tone: "error",
        title: t(
          file.size > 10 * 1024 * 1024 ? "resume.tooLarge" : "resume.pdfOnly",
        ),
      });
      return;
    }
    upload.mutate(file);
  };
  const download = async () => {
    try {
      const blob = await repositories.candidate.downloadOwnResume();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = profile.data?.resume?.originalFilename ?? "resume.pdf";
      link.click();
      URL.revokeObjectURL(url);
    } catch {
      toast.showToast({ tone: "error", title: t("resume.downloadError") });
    }
  };

  return (
    <CandidateWorkspaceLayout>
      <section className="workspace-page" aria-labelledby="resume-page-title">
        <header className="workspace-page__header">
          <p className="auth-eyebrow">{t("workspace.title")}</p>
          <h1 id="resume-page-title">{t("workspace.nav.resume")}</h1>
          <p>{t("workspace.resumeDescription")}</p>
        </header>
        {profile.isPending ? (
          <Skeleton className="detail-skeleton__body" />
        ) : null}
        {profile.isError ? (
          <ErrorState
            title={t("profile.loadErrorTitle")}
            description={t("profile.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void profile.refetch()}
          />
        ) : null}
        {profile.data ? (
          <section className="profile-section">
            <div className="profile-section__body">
              <input
                ref={fileRef}
                className="visually-hidden"
                type="file"
                accept="application/pdf,.pdf"
                onChange={(event) => chooseFile(event.target.files)}
              />
              {profile.data.resume ? (
                <div className="resume-row">
                  <FileText size={26} aria-hidden="true" />
                  <div>
                    <strong>{profile.data.resume.originalFilename}</strong>
                    <span>
                      {formatBytes(profile.data.resume.sizeBytes)} ·{" "}
                      {formatDate(
                        profile.data.resume.uploadedAt,
                        i18n.language,
                      )}{" "}
                      · {t(`resume.statuses.${profile.data.resume.status}`)}
                    </span>
                  </div>
                  <div>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      leadingIcon={<Download size={15} />}
                      onClick={() => void download()}
                    >
                      {t("resume.download")}
                    </Button>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      leadingIcon={<Upload size={15} />}
                      isLoading={upload.isPending}
                      onClick={() => fileRef.current?.click()}
                    >
                      {t("resume.replace")}
                    </Button>
                    <Button
                      type="button"
                      variant="danger"
                      size="sm"
                      leadingIcon={<Trash2 size={15} />}
                      isLoading={remove.isPending}
                      onClick={() =>
                        window.confirm(t("resume.deleteConfirm")) &&
                        remove.mutate()
                      }
                    >
                      {t("resume.delete")}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="resume-upload">
                  <FileText size={28} aria-hidden="true" />
                  <div>
                    <strong>{t("resume.emptyTitle")}</strong>
                    <p>{t("resume.emptyDescription")}</p>
                  </div>
                  <Button
                    type="button"
                    leadingIcon={<Upload size={17} />}
                    isLoading={upload.isPending}
                    onClick={() => fileRef.current?.click()}
                  >
                    {t("resume.upload")}
                  </Button>
                </div>
              )}
            </div>
          </section>
        ) : null}
      </section>
    </CandidateWorkspaceLayout>
  );
}

function formatBytes(value: number) {
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}
function formatDate(value: string, language: string) {
  return new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
    new Date(value),
  );
}
