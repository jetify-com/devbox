import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const nixhubSidebar: SidebarsConfig = {
  sidebar: [
    {
      type: "doc",
      label: "Get a Package",
      id: "nixhub/get-a-package",
    },
    {
      type: "doc",
      label: "Search Packages",
      id: "nixhub/search-packages",
    },
    {
      type: "doc",
      label: "Resolve a Package",
      id: "nixhub/resolve-a-package-version",
    },
  ],
};

export default nixhubSidebar.sidebar;
