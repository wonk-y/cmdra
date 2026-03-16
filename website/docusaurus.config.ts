import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const organizationName = process.env.ORGANIZATION_NAME ?? 'your-github-user';
const projectName = process.env.PROJECT_NAME ?? 'your-repo';
const deploymentBranch = process.env.DEPLOYMENT_BRANCH ?? 'gh-pages';
const siteUrl = process.env.DOCS_URL ?? 'http://localhost';
const siteBaseUrl = process.env.DOCS_BASE_URL ?? '/';

const config: Config = {
  title: 'CmdAgent Docs',
  tagline: 'Operate cmdagentd across Linux, macOS, and Windows with mTLS, the CLI, the TUI, SDKs, Robot Framework, and Ansible.',
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
    image: 'img/cmdagent-social-card.svg',
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'CmdAgent',
      logo: {
        alt: 'CmdAgent',
        src: 'img/cmdagent-logo.svg',
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
          to: `${siteBaseUrl}docs/cli/cmdagentctl`,
          label: 'CLI',
          position: 'left',
        },
        {
          to: `${siteBaseUrl}docs/cli/cmdagentui`,
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
            {label: 'cmdagentctl', to: `${siteBaseUrl}docs/cli/cmdagentctl`},
            {label: 'cmdagentui', to: `${siteBaseUrl}docs/cli/cmdagentui`},
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
      copyright: `Copyright © ${new Date().getFullYear()} CmdAgent documentation. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.nightOwl,
      additionalLanguages: ['bash', 'powershell', 'python', 'go', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
