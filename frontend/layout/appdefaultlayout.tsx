import { getCurrentHash } from 'f61ui/browserutils';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { NavLink } from 'f61ui/component/navigation';
import { DefaultLayout } from 'f61ui/layout/defaultlayout';
import { jsxChildType } from 'f61ui/types';
import { RootFolderId } from 'generated/stoserver/stoservertypes_types';
import { version } from 'generated/version';
import * as React from 'react';
import {
	browseRoute,
	clientsRoute,
	nodesRoute,
	replicationPoliciesRoute,
	serverInfoRoute,
	volumesAndMountsRoute,
} from 'routes';

interface AppDefaultLayoutProps {
	title: string;
	breadcrumbs: Breadcrumb[];
	children: jsxChildType;
}

// app's default layout uses the default layout with props that are common to the whole app
export class AppDefaultLayout extends React.Component<AppDefaultLayoutProps, {}> {
	render() {
		const hash = getCurrentHash();

		const navLinks: NavLink[] = [
			{
				title: 'Browse',
				url: browseRoute.buildUrl({ dir: RootFolderId, v: '' }),
				active: browseRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Volumes & mounts',
				url: volumesAndMountsRoute.buildUrl({}),
				active: volumesAndMountsRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Nodes',
				url: nodesRoute.buildUrl({}),
				active: nodesRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Clients',
				url: clientsRoute.buildUrl({}),
				active: clientsRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Replication policies',
				url: replicationPoliciesRoute.buildUrl({}),
				active: replicationPoliciesRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Server info',
				url: serverInfoRoute.buildUrl({}),
				active: serverInfoRoute.matchUrl(hash) !== null,
			},
		];

		return (
			<DefaultLayout
				appName="Varasto"
				appHomepage="https://github.com/function61/varasto"
				navLinks={navLinks}
				logoUrl={browseRoute.buildUrl({ dir: RootFolderId, v: '' })}
				breadcrumbs={this.props.breadcrumbs.concat({ title: this.props.title })}
				content={this.props.children}
				version={version}
				pageTitle={this.props.title}
			/>
		);
	}
}
