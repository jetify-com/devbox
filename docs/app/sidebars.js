/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
    // By default, Docusaurus generates a sidebar from the docs folder structure
    // tutorialSidebar: [{ type: 'autogenerated', dirName: '.' }],

    // But you can create a sidebar manually
    tutorialSidebar: [{
        type: 'doc',
        id: 'index'
    }, {
        type: 'doc',
        id: 'installing_devbox'
    }, {
        type: 'doc',
        id: 'quickstart'
    }, {
        type: 'category',
        label: 'CLI Reference',
        link: { type: 'doc', id: 'cli_reference/devbox' },
        collapsed: true,
        items: [{
            type: 'doc',
            id: 'cli_reference/devbox_add',
            label: 'devbox add'
        }, {
            type: 'doc',
            id: 'cli_reference/devbox_build',
            label: 'devbox build'
        }, {
            type: 'doc',
            id: 'cli_reference/devbox_init',
            label: 'devbox init'
        }, {
            type: 'doc',
            id: 'cli_reference/devbox_plan',
            label: 'devbox plan'
        }, {
            type: 'doc',
            id: 'cli_reference/devbox_rm',
            label: 'devbox rm'
        }, {
            type: 'doc',
            id: 'cli_reference/devbox_shell',
            label: 'devbox shell'
        }, {
            type: 'doc',
            id: 'cli_reference/devbox_version',
            label: 'devbox version'
        }]
    }, {
        type: 'doc',
        id: 'configuration'
    }, {
        type: 'category',
        label: 'Language Detection',
        collapsed: false,
        items: [{
            type: 'doc',
            id: 'language_support/csharp'
        }, {
            type: 'doc',
            id: 'language_support/go'
        }, {
            type: 'doc',
            id: 'language_support/haskell'
        }, {
            type: 'doc',
            id: 'language_support/java'
        }, {
            type: 'doc',
            id: 'language_support/nginx'
        }, {
            type: 'doc',
            id: 'language_support/nodejs'
        }, {
            type: 'doc',
            id: 'language_support/php'
        }, {
            type: 'doc',
            id: 'language_support/python'
        }, {
            type: 'doc',
            id: 'language_support/ruby'
        }, {
          type: 'doc',
            id: 'language_support/rust'
        }, {
            type: 'doc',
            id: 'language_support/zig'
        }]
    }],

};

module.exports = sidebars;