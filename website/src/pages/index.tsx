import clsx from 'clsx';
import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import useBaseUrl from '@docusaurus/useBaseUrl';
import type {ReactElement} from 'react';

import styles from './index.module.css';

const sections = [
  {
    title: 'Install the daemon',
    description:
      'Set up cmdrad in foreground or service mode on Linux, macOS, and Windows, with mTLS and CN-based access control.',
    to: 'docs/install/linux',
    eyebrow: 'Operations',
  },
  {
    title: 'Operate with cmdractl',
    description:
      'Run commands, attach to shell sessions, inspect metadata, and move files through the operator CLI.',
    to: 'docs/cli/cmdractl',
    eyebrow: 'CLI',
  },
  {
    title: 'Operate with cmdraui',
    description:
      'Use the Bubble Tea terminal UI to inspect executions and transfers, launch new work, and attach to live sessions.',
    to: 'docs/cli/cmdraui',
    eyebrow: 'TUI',
  },
  {
    title: 'Build against the SDKs',
    description:
      'Use the Go and Python client libraries for synchronous and asynchronous execution and transfer workflows.',
    to: 'docs/sdk/go',
    eyebrow: 'SDKs',
  },
  {
    title: 'Integrate with test and automation tools',
    description:
      'Use the Python wrapper layers with Robot Framework and Ansible without building your own transport glue.',
    to: 'docs/integrations/robot-framework',
    eyebrow: 'Integrations',
  },
];

function Hero() {
  const docsUrl = useBaseUrl('/docs/');

  return (
    <section className={styles.hero}>
      <div className={clsx('container', styles.heroGrid)}>
        <div className={styles.heroCopy}>
          <p className={styles.kicker}>Cmdra Documentation</p>
          <Heading as="h1" className={styles.title}>
            Remote execution docs for the daemon, the CLI, and every supported client surface.
          </Heading>
          <p className={styles.subtitle}>
            Cmdra exposes argv commands, shell commands, persistent shell sessions, output replay,
            and file transfer over gRPC with mutual TLS. This site documents installation and usage
            across Linux, macOS, Windows, the CLI, the TUI, Go, Python, Robot Framework, and Ansible.
          </p>
          <div className={styles.heroActions}>
            <Link className="button button--primary button--lg" to={docsUrl}>
              Read the docs
            </Link>
          </div>
        </div>
        <div className={styles.heroPanel}>
          <div className={styles.panelHeader}>Quick start</div>
          <pre className={styles.commandBlock}>
            <code>{`./scripts/generate-dev-certs.sh dev/certs
./cmdrad run --config ./dev/cmdrad.json
./cmdractl --address 127.0.0.1:8443 \
  --ca dev/certs/ca.crt \
  --cert dev/certs/client-a.crt \
  --key dev/certs/client-a.key list
./cmdraui --address 127.0.0.1:8443 \
  --ca dev/certs/ca.crt \
  --cert dev/certs/client-a.crt \
  --key dev/certs/client-a.key`}</code>
          </pre>
        </div>
      </div>
    </section>
  );
}

function SectionCards() {
  return (
    <section className={styles.sectionShell}>
      <div className="container">
        <div className={styles.sectionHeading}>
          <p className={styles.sectionKicker}>Coverage</p>
          <Heading as="h2">Operational use notes</Heading>
        </div>
        <div className={styles.cardGrid}>
          {sections.map((section) => (
            <Link key={section.title} className={styles.card} to={section.to}>
              <span className={styles.cardEyebrow}>{section.eyebrow}</span>
              <Heading as="h3" className={styles.cardTitle}>
                {section.title}
              </Heading>
              <p className={styles.cardDescription}>{section.description}</p>
              <span className={styles.cardAction}>Open guide</span>
            </Link>
          ))}
        </div>
      </div>
    </section>
  );
}

export default function Home(): ReactElement {
  return (
    <Layout
      title="Documentation"
      description="Installation and usage guides for cmdrad, cmdractl, cmdraui, the Go SDK, the Python SDK, Robot Framework, and Ansible.">
      <Hero />
      <SectionCards />
    </Layout>
  );
}
