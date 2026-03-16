import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Install cmdrad',
      items: [
        'install/certs',
        'install/linux',
        'install/macos',
        'install/windows',
      ],
    },
    {
      type: 'category',
      label: 'Use the CLI and TUI',
      collapsed: false,
      items: ['cli/cmdractl', 'cli/cmdraui', 'cli/pty-attach-checklist'],
    },
    {
      type: 'category',
      label: 'Client Libraries',
      items: ['sdk/go', 'sdk/python'],
    },
    {
      type: 'category',
      label: 'Integrations',
      items: ['integrations/robot-framework', 'integrations/ansible'],
    },
  ],
};

export default sidebars;
