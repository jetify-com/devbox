// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

import { themes } from "prism-react-renderer";

const codeTheme = { light: themes.github, dark: themes.dracula };

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Jetify Docs",
  tagline: "Instant, easy, and predictable shells and containers",
  url: "https://www.jetify.com",
  baseUrl: "/",
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",
  favicon: "img/favicon.ico",
  trailingSlash: true,

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "jetify-com", // Usually your GitHub org/user name.
  projectName: "devbox", // Usually your repo name.

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  markdown: {
    mermaid: true,
  },
  themes: ["docusaurus-theme-openapi-docs", "@docusaurus/theme-mermaid"],
  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },
  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: "/devbox/docs",
          sidebarPath: require.resolve("./sidebars.js"),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          docItemComponent: "@theme/ApiItem",
          editUrl: "https://github.com/jetify-com/devbox/tree/main/docs/app/",
        },
        blog: false,
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },

        gtag: {
          trackingID: "G-PL4J94CXFK",
          anonymizeIP: true,
        },
      }),
    ],
  ],

  plugins: [
    [
      "@docusaurus/plugin-content-docs",
      {
        id: "cloud",
        path: "cloud",
        routeBasePath: "/cloud/docs",
        sidebarPath: require.resolve("./cloud_sidebars.js"),
      },
    ],
    [
      "docusaurus-plugin-openapi-docs",
      {
        id: "api",
        docsPluginId: "classic",
        config: {
          nixhub: {
            specPath: "specs/nixhub.yaml",
            outputDir: "docs/nixhub",
          },
        },
      },
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        logo: {
          alt: "Devbox",
          src: "img/devbox_logo_light.svg",
          srcDark: "img/devbox_logo_dark.svg",
          href: "/devbox/docs",
        },
        items: [
          {
            to: "cloud/docs",
            label: "Jetify Cloud",
            className: "header-text-link",
            position: "left",
          },
          {
            to: "/devbox/docs/nixhub",
            label: "NixHub API",
            className: "header-text-link",
            position: "left",
          },
          {
            href: "https://discord.gg/jetify",
            // label: 'Discord',
            className: "header-discord-link",
            position: "right",
          },
          {
            href: "https://github.com/jetify-com/devbox",
            // label: 'GitHub',
            className: "header-github-link",
            position: "right",
          },
        ],
      },
      footer: {
        links: [
          {
            title: "Jetify",
            items: [
              {
                label: "Jetify",
                href: "https://www.jetify.com",
              },
              {
                label: "Blog",
                href: "https://www.jetify.com/blog",
              },
              {
                label: "Jetify Cloud",
                href: "https://cloud.jetify.com",
              },
            ],
          },
          {
            title: "Devbox",
            items: [
              {
                label: "Home",
                to: "https://www.jetify.com/devbox",
              },
              {
                label: "Docs",
                to: "https://www.jetify.com/devbox/docs/",
              },
            ],
          },

          {
            title: "Community",
            items: [
              {
                label: "Github",
                href: "https://github.com/jetify-com",
              },
              {
                label: "Twitter",
                href: "https://twitter.com/jetify_com",
              },
              {
                href: "https://discord.gg/jetify",
                label: "Discord",
              },
              {
                href: "https://www.youtube.com/channel/UC7FwfJZbunZR2s-jG79vuTQ",
                label: "Youtube",
              },
            ],
          },
        ],
        style: "dark",
        copyright: `Copyright © ${new Date().getFullYear()} Jetify, Inc.`,
      },
      colorMode: {
        respectPrefersColorScheme: true,
      },
      algolia: {
        appId: "J1RTMNIB0R",
        apiKey: "b1bcbf465b384ccd6d986e85d6a62c28",
        indexName: "jetpack",
        searchParameters: {},
      },
      prism: {
        theme: codeTheme.light,
        darkTheme: codeTheme.dark,
        additionalLanguages: ["bash", "json"],
      },
    }),
};

export default config;
