// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightLlmsTxt from "starlight-llms-txt";

// https://astro.build/config
export default defineConfig({
  site: "https://platforma-dev.github.io",
  base: "/platforma",
  integrations: [
    starlight({
      title: "platforma",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/platforma-dev/platforma",
        },
      ],
      sidebar: [
        {
          slug: "getting-started",
        },
        {
          label: "Packages",
          items: [
            "packages/application",
            "packages/database",
            "packages/httpserver",
            "packages/log",
            "packages/log2",
            "packages/queue",
            "packages/scheduler",
            "packages/auth",
          ],
        },
        {
          slug: "cli",
        },
        {
          label: "AI docs",
          items: [
            {
              label: "llms.txt",
              link: "/llms.txt",
            },
            {
              label: "llms-small.txt",
              link: "/llms-small.txt",
            },
            {
              label: "llms-full.txt",
              link: "/llms-full.txt",
            },
          ],
        },
      ],
      plugins: [
        starlightLlmsTxt({
          projectName: "Platforma",
        }),
      ],
    }),
  ],
});
