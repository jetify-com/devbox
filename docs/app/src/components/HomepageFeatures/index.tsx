import React from 'react';
import clsx from 'clsx';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  // url: string;
  Svg: React.ComponentType<React.ComponentProps<'svg'>>;
  description: JSX.Element;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Isolated local shells on your laptop',
    Svg: require('@site/static/img/nix-term-light.svg').default,
    // url: `/docs`,
    description: (
      <>
        Start by defining the list of packages required by your project, and Devbox will create an isolated dev environment on your local machine, no Docker required. 
      </>
    ),
  },
  {
    title: 'Over 80,000 packages, powered by Nix',
    Svg: require('@site/static/img/nix_snowflake_light.svg').default,
    // url: 'http://github.com/jetpack-io/devbox',
    description: (
      <>
        Devbox provides an approachable, intuitive interface for creating isolated shells with the Nix Package Manager. Create an environment for any project, on any machine. 
      </>
    ),
  },
  {
    title: 'Open Source and Community Driven',
    Svg: require('@site/static/img/github-light.svg').default,
    // url: 'https://discord.com/invite/agbskCJXk2',
    description: (
      <>
        Devbox is an open source project built by <b><a href="https://jetpack.io"> Jetpack.io</a></b> with support from the community. Join the core team and thousands of developers who love Devbox on <b><a href="http://discord.gg/jetpack-io">Discord</a></b>. 
      </>
    ),
  },
];

function Feature({title, Svg, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className={styles.featureItem}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): JSX.Element {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className={`row ${styles.featureRow}`}>
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
