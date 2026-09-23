import { Link, Route, Routes } from "react-router-dom";
import { FoundationPage } from "./pages/FoundationPage";
export default function App() {
  return (
    <Routes>
      <Route path="/" element={<FoundationPage />} />
      <Route
        path="*"
        element={
          <main>
            <h1>Page not found</h1>
            <Link to="/">Return home</Link>
          </main>
        }
      />
    </Routes>
  );
}
