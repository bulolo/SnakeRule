import { defineConfig } from "orval";

export default defineConfig({
  snakeRule: {
    input: "./openapi.json",
    output: {
      mode: "tags",
      target: "./src/lib/sdk/sdk.ts",
      // SnakeRule's admin calls the generated operations imperatively; it does not use TanStack Query.
      client: "fetch",
      clean: true,
      override: {
        fetch: {
          includeHttpResponseReturnType: false,
        },
        mutator: {
          path: "./src/lib/api/http.ts",
          name: "customFetch",
        },
      },
    },
  },
});
