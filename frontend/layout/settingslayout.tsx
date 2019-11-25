import { getCurrentHash } from 'f61ui/browserutils';
import { Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { NavLink, renderNavLink } from 'f61ui/component/navigation';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import * as r from 'routes';

interface SettingsLayoutProps {
	title: string;
	breadcrumbs: Breadcrumb[];
	children: React.ReactNode;
}

export class SettingsLayout extends React.Component<SettingsLayoutProps, {}> {
	render() {
		const hash = getCurrentHash();

		const settingsLinks: NavLink[] = [
			{
				title: 'Server info & health',
				glyphicon: 'dashboard',
				url: r.serverInfoRoute.buildUrl({}),
				active: r.serverInfoRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Volumes & mounts',
				glyphicon: 'hdd',
				url: r.volumesAndMountsRoute.buildUrl({ view: '' }),
				active: r.volumesAndMountsRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Scheduled jobs',
				glyphicon: 'time',
				url: r.scheduledJobsRoute.buildUrl({}),
				active: r.scheduledJobsRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Backups',
				glyphicon: 'cloud-upload',
				url: r.metadataBackupRoute.buildUrl({ v: '' }),
				active: r.metadataBackupRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Users',
				glyphicon: 'user',
				url: r.usersRoute.buildUrl({}),
				active: r.usersRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Logs',
				glyphicon: 'list-alt',
				url: r.logsRoute.buildUrl({}),
				active: r.logsRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Nodes',
				glyphicon: 'th-large',
				url: r.nodesRoute.buildUrl({}),
				active: r.nodesRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Replication policies',
				glyphicon: 'retweet',
				url: r.replicationPoliciesRoute.buildUrl({}),
				active: r.replicationPoliciesRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Content metadata',
				glyphicon: 'book',
				url: r.contentMetadataRoute.buildUrl({}),
				active: r.contentMetadataRoute.matchUrl(hash) !== null,
			},
			{
				title: 'FUSE server & network folders',
				glyphicon: 'folder-open',
				url: r.fuseServerRoute.buildUrl({}),
				active: r.fuseServerRoute.matchUrl(hash) !== null,
			},
		];

		return (
			<AppDefaultLayout
				title={this.props.title}
				breadcrumbs={this.props.breadcrumbs.concat({
					url: r.serverInfoRoute.buildUrl({}),
					title: 'Settings',
				})}
				children={
					<div className="row">
						<div className="col-md-3">
							<Panel heading="Settings">
								<ul className="nav nav-pills nav-stacked">
									{settingsLinks.map(renderNavLink)}
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
