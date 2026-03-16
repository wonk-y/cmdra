import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const organizationName = process.env.ORGANIZATION_NAME ?? 'your-github-user';
const projectName = process.env.PROJECT_NAME ?? 'your-repo';
const deploymentBranch = process.env.DEPLOYMENT_BRANCH ?? 'gh-pages';
const siteUrl = process.env.DOCS_URL ?? 'http://localhost';
const siteBaseUrl = process.env.DOCS_BASE_URL ?? '/';

const config: Config = {
  title: 'Cmdra Docs',
  tagline: 'Operate cmdrad across Linux, macOS, and Windows with mTLS, the CLI, the TUI, SDKs, Robot Framework, and Ansible.',
  favicon: 'img/favicon.ico',
  url: siteUrl,
  baseUrl: siteBaseUrl,
  organizationName,
  projectName,
  deploymentBranch,
  trailingSlash: false,
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'throw',
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },
  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: 'docs',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],
  themeConfig: {
    image: 'img/cmdra-social-card.svg',
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Cmdra',
      logo: {
        alt: 'Cmdra',
        src: 'img/cmdra-logo.svg',
        href: siteBaseUrl,
      },
      items: [
        {
          to: `${siteBaseUrl}docs/`,
          label: 'Docs',
          position: 'left',
        },
        {
          to: `${siteBaseUrl}docs/install/linux`,
          label: 'Install',
          position: 'left',
        },
        {
          to: `${siteBaseUrl}docs/cli/cmdractl`,
          label: 'CLI',
          position: 'left',
        },
        {
          to: `${siteBaseUrl}docs/cli/cmdraui`,
          label: 'TUI',
          position: 'left',
        },
        {
          to: `${siteBaseUrl}docs/sdk/go`,
          label: 'SDKs',
          position: 'left',
        },
        {
          to: `${siteBaseUrl}docs/integrations/robot-framework`,
          label: 'Integrations',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Install',
          items: [
            {label: 'Linux', to: `${siteBaseUrl}docs/install/linux`},
            {label: 'macOS', to: `${siteBaseUrl}docs/install/macos`},
            {label: 'Windows', to: `${siteBaseUrl}docs/install/windows`},
          ],
        },
        {
          title: 'Use',
          items: [
            {label: 'cmdractl', to: `${siteBaseUrl}docs/cli/cmdractl`},
            {label: 'cmdraui', to: `${siteBaseUrl}docs/cli/cmdraui`},
            {label: 'Go SDK', to: `${siteBaseUrl}docs/sdk/go`},
            {label: 'Python SDK', to: `${siteBaseUrl}docs/sdk/python`},
          ],
        },
        {
          title: 'Integrate',
          items: [
            {label: 'Robot Framework', to: `${siteBaseUrl}docs/integrations/robot-framework`},
            {label: 'Ansible', to: `${siteBaseUrl}docs/integrations/ansible`},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Cmdra documentation. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.nightOwl,
      additionalLanguages: ['bash', 'powershell', 'python', 'go', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
