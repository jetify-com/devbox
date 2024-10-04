import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const nixhubSidebar: SidebarsConfig = {
  sidebar: [
    {
        type: "doc",
        id: "index",
      },
    {
      type: "doc",
      label: "Get a Package",
      id: "get-a-package",
    },
    {
      type: "doc",
      label: "Search Packages",
      id: "search-packages",
    },
    {
      type: "doc",
      label: "Resolve a Package",
      id: "resolve-a-package-version",
    },
  ],
};

export default nixhubSidebar.sidebar;