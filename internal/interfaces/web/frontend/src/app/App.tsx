import { ThemeProvider } from "../features/theme/ThemeProvider";
import { WorkbenchApp } from "./WorkbenchApp";

export function App() {
  return (
    <ThemeProvider>
      <WorkbenchApp />
    </ThemeProvider>
  );
}
