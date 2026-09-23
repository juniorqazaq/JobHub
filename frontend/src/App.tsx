import { lazy, Suspense, useEffect } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";

const HomePage = lazy(() => import("./pages/HomePage").then((module) => ({ default: module.HomePage })));
const JobsPage = lazy(() => import("./pages/JobsPage").then((module) => ({ default: module.JobsPage })));
const JobDetailPage = lazy(() => import("./pages/JobDetailPage").then((module) => ({ default: module.JobDetailPage })));

const ComponentShowcasePage = lazy(() =>
  import("./pages/ComponentShowcasePage").then((module) => ({
    default: module.ComponentShowcasePage,
  })),
);

export default function App() {
  return (
    <>
      <ScrollToTop />
      <Suspense fallback={<div className="route-loading" aria-hidden="true" />}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/jobs" element={<JobsPage />} />
          <Route path="/jobs/:jobId" element={<JobDetailPage />} />
          <Route path="/dev/ui" element={<ComponentShowcasePage />} />
          <Route path="/dev/components" element={<Navigate to="/dev/ui" replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
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
