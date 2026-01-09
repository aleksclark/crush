import { defineConfig } from "@microsoft/tui-test";

export default defineConfig({
  retries: 2,
  trace: false,
  timeout: 30000,
});
