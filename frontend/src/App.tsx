import { lazy, Suspense, useEffect } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { JobHubLoader } from "./components/ui/JobHubLoader";

const HomePage = lazy(() => import("./pages/HomePage").then((module) => ({ default: module.HomePage })));
const JobsPage = lazy(() => import("./pages/JobsPage").then((module) => ({ default: module.JobsPage })));
const JobDetailPage = lazy(() => import("./pages/JobDetailPage").then((module) => ({ default: module.JobDetailPage })));
const LoginPage = lazy(() => import("./pages/LoginPage").then((module) => ({ default: module.LoginPage })));
const RegisterPage = lazy(() => import("./pages/RegisterPage").then((module) => ({ default: module.RegisterPage })));
const AccountPage = lazy(() => import("./pages/AccountPage").then((module) => ({ default: module.AccountPage })));
const EmployerPage = lazy(() => import("./pages/EmployerPage").then((module) => ({ default: module.EmployerPage })));
const EmployerVacanciesPage = lazy(() => import("./pages/EmployerVacanciesPage").then((module) => ({ default: module.EmployerVacanciesPage })));
const EmployerVacancyFormPage = lazy(() => import("./pages/EmployerVacancyFormPage").then((module) => ({ default: module.EmployerVacancyFormPage })));
const AdminPage = lazy(() => import("./pages/AdminPage").then((module) => ({ default: module.AdminPage })));
const ProtectedRoute = lazy(() => import("./pages/ProtectedRoute").then((module) => ({ default: module.ProtectedRoute })));
const ProfilePage = lazy(() => import("./pages/ProfilePage").then((module) => ({ default: module.ProfilePage })));
const SavedJobsPage = lazy(() => import("./pages/SavedJobsPage").then((module) => ({ default: module.SavedJobsPage })));
const ApplicationsPage = lazy(() => import("./pages/ApplicationsPage").then((module) => ({ default: module.ApplicationsPage })));
const EmployerApplicantsPage = lazy(() => import("./pages/EmployerApplicantsPage").then((module) => ({ default: module.EmployerApplicantsPage })));
const ResumePage = lazy(() => import("./pages/ResumePage").then((module) => ({ default: module.ResumePage })));
const NotFoundPage = lazy(() => import("./pages/NotFoundPage").then((module) => ({ default: module.NotFoundPage })));

const ComponentShowcasePage = lazy(() =>
  import("./pages/ComponentShowcasePage").then((module) => ({
    default: module.ComponentShowcasePage,
  })),
);

export default function App() {
  return (
    <>
      <ScrollToTop />
      <Suspense fallback={<div className="route-loading"><JobHubLoader /></div>}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/jobs" element={<JobsPage />} />
          <Route path="/jobs/:jobId" element={<JobDetailPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/account" element={<ProtectedRoute><AccountPage /></ProtectedRoute>} />
          <Route path="/workspace" element={<ProtectedRoute role="job_seeker"><Navigate to="/profile" replace /></ProtectedRoute>} />
          <Route path="/profile" element={<ProtectedRoute role="job_seeker"><ProfilePage /></ProtectedRoute>} />
          <Route path="/saved" element={<ProtectedRoute role="job_seeker"><SavedJobsPage /></ProtectedRoute>} />
          <Route path="/applications" element={<ProtectedRoute role="job_seeker"><ApplicationsPage /></ProtectedRoute>} />
          <Route path="/resume" element={<ProtectedRoute role="job_seeker"><ResumePage /></ProtectedRoute>} />
          <Route path="/preferences" element={<ProtectedRoute role="job_seeker"><Navigate to="/resume#preferences" replace /></ProtectedRoute>} />
          <Route path="/employer" element={<ProtectedRoute role="employer"><EmployerPage /></ProtectedRoute>} />
          <Route path="/employer/vacancies" element={<ProtectedRoute role="employer"><EmployerVacanciesPage /></ProtectedRoute>} />
          <Route path="/employer/vacancies/new" element={<ProtectedRoute role="employer"><EmployerVacancyFormPage /></ProtectedRoute>} />
          <Route path="/employer/vacancies/:jobId/edit" element={<ProtectedRoute role="employer"><EmployerVacancyFormPage /></ProtectedRoute>} />
          <Route path="/employer/vacancies/:jobId/applicants" element={<ProtectedRoute role="employer"><EmployerApplicantsPage /></ProtectedRoute>} />
          <Route path="/admin" element={<ProtectedRoute role="admin"><AdminPage /></ProtectedRoute>} />
          <Route path="/dev/ui" element={<ComponentShowcasePage />} />
          <Route path="/dev/components" element={<Navigate to="/dev/ui" replace />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </Suspense>
    </>
  );
}

function ScrollToTop() {
  const { pathname } = useLocation();
  useEffect(() => {
    window.scrollTo({ top: 0, left: 0 });
  }, [pathname]);
  return null;
}
