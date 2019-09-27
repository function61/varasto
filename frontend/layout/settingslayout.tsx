import { getCurrentHash } from 'f61ui/browserutils';
import { Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { NavLink } from 'f61ui/component/navigation';
import { jsxChildType } from 'f61ui/types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import {
	clientsRoute,
	contentMetadataRoute,
	encryptionKeysRoute,
	fuseServerRoute,
	healthRoute,
	logsRoute,
	nodesRoute,
	replicationPoliciesRoute,
	serverInfoRoute,
	usersRoute,
	volumesAndMountsRoute,
} from 'routes';

interface SettingsLayoutProps {
	title: string;
	breadcrumbs: Breadcrumb[];
	children: jsxChildType;
}

export class SettingsLayout extends React.Component<SettingsLayoutProps, {}> {
	render() {
		const hash = getCurrentHash();

		const settingsLinks: NavLink[] = [
			{
				title: 'Server info',
				url: serverInfoRoute.buildUrl({}),
				active: serverInfoRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Health',
				url: healthRoute.buildUrl({}),
				active: healthRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Logs',
				url: logsRoute.buildUrl({}),
				active: logsRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Volumes & mounts',
				url: volumesAndMountsRoute.buildUrl({}),
				active: volumesAndMountsRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Users',
				url: usersRoute.buildUrl({}),
				active: usersRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Encryption keys',
				url: encryptionKeysRoute.buildUrl({}),
				active: encryptionKeysRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Clients',
				url: clientsRoute.buildUrl({}),
				active: clientsRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Nodes',
				url: nodesRoute.buildUrl({}),
				active: nodesRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Replication policies',
				url: replicationPoliciesRoute.buildUrl({}),
				active: replicationPoliciesRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Content metadata',
				url: contentMetadataRoute.buildUrl({}),
				active: contentMetadataRoute.matchUrl(hash) !== null,
			},
			{
				title: 'FUSE server & network folders',
				url: fuseServerRoute.buildUrl({}),
				active: fuseServerRoute.matchUrl(hash) !== null,
			},
		];

		return (
			<AppDefaultLayout
				title={this.props.title}
				breadcrumbs={this.props.breadcrumbs.concat({
					url: serverInfoRoute.buildUrl({}),
					title: 'Settings',
				})}
				children={
					<div className="row">
						<div className="col-md-3">
							<Panel heading="Settings">
								<ul className="nav nav-pills nav-stacked">
									{settingsLinks.map((l) => (
										<li
											role="presentation"
											className={l.active ? 'active' : ''}>
											<a href={l.url}>{l.title}</a>
										</li>
									))}
								</ul>
							</Panel>
						</div>
						<div className="col-md-9">{this.props.children}</div>
					</div>
				}
			/>
		);
	}
}
