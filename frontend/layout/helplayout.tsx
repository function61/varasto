import { getCurrentLocation } from 'f61ui/browserutils';
import { Panel, GlyphiconIcon } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { NavLink, renderNavLink } from 'f61ui/component/navigation';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import * as r from 'generated/stoserver/stoserverui_uiroutes';

interface HelpLayoutProps {
	title: string;
	breadcrumbs: Breadcrumb[];
	children: React.ReactNode;
}

export class HelpLayout extends React.Component<HelpLayoutProps, {}> {
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

		const helpDocs: NavLink[] = [
			mkLink('Getting started', 'home', r.gettingStartedUrl({ section: 'welcome' })),
			mkLink('Download client app', 'download-alt', r.downloadClientAppUrl()),
		];

		return (
			<AppDefaultLayout
				title={this.props.title}
				breadcrumbs={this.props.breadcrumbs.concat({
					url: r.gettingStartedUrl({ section: 'welcome' }),
					title: 'Help',
				})}
				children={
					<div className="row">
						<div className="col-md-3">
							<Panel heading="Help">
								<ul className="nav nav-pills nav-stacked">
									{helpDocs.map(renderNavLink)}
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
