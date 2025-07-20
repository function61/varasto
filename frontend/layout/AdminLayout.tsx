import { getCurrentLocation } from 'f61ui/browserutils';
import { GlyphiconIcon, Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { NavLink, renderNavLink } from 'f61ui/component/navigation';
import * as r from 'generated/frontend_uiroutes';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface AdminLayoutProps {
	title: string;
	breadcrumbs: Breadcrumb[];
	children: React.ReactNode;
}

export class AdminLayout extends React.Component<AdminLayoutProps, {}> {
	render() {
		const currLoc = getCurrentLocation();

		function mkLink(title: string, icon: GlyphiconIcon, url: string): NavLink {
			return {
				title,
				glyphicon: icon,
				url,
				active: url === currLoc,
			};
		}

		const settingsLinks: NavLink[] = [
			mkLink('Health & server info', 'dashboard', r.serverInfoUrl()),
			mkLink('Volumes & mounts', 'hdd', r.volumesUrl()),
			mkLink('Subsystems', 'tasks', r.subsystemsUrl()),
			mkLink('Scheduled jobs', 'time', r.scheduledJobsUrl()),
			mkLink('Metadata backup', 'cloud-upload', r.metadataBackupUrl({ view: '' })),
			mkLink('Users', 'user', r.usersUrl()),
			mkLink('Metrics', 'stats', r.metricsUrl()),
			mkLink('Logs', 'list-alt', r.logsUrl()),
			mkLink('Servers', 'th-large', r.nodesUrl()),
			mkLink('Replication policies', 'retweet', r.replicationPoliciesUrl()),
			mkLink('Content metadata', 'book', r.contentMetadataUrl()),
			mkLink('FUSE server & network folders', 'folder-open', r.fuseServerUrl()),
		];

		return (
			<AppDefaultLayout
				title={this.props.title}
				breadcrumbs={this.props.breadcrumbs.concat({
					url: r.serverInfoUrl(),
					title: 'Admin',
				})}
				children={
					<div className="row">
						<div className="col-md-3">
							<Panel heading="Admin">
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
