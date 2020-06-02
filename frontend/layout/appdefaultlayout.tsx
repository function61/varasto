import { getCurrentLocation } from 'f61ui/browserutils';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { NavLink } from 'f61ui/component/navigation';
import { GlyphiconIcon } from 'f61ui/component/bootstrap';
import { globalConfig } from 'f61ui/globalconfig';
import { DefaultLayout } from 'f61ui/layout/defaultlayout';
import { RootFolderId } from 'generated/stoserver/stoservertypes_types';
import { version } from 'generated/version';
import * as React from 'react';
import { browseUrl, gettingStartedUrl, serverInfoUrl } from 'generated/frontend_uiroutes';

interface AppDefaultLayoutProps {
	title: string;
	titleElem?: React.ReactNode;
	breadcrumbs: Breadcrumb[];
	children: React.ReactNode;
}

// app's default layout uses the default layout with props that are common to the whole app
export class AppDefaultLayout extends React.Component<AppDefaultLayoutProps, {}> {
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

		const navLinks: NavLink[] = [
			mkLink('Browse', 'folder-open', browseUrl({ dir: RootFolderId })),
			mkLink('Help', 'book', gettingStartedUrl({ section: 'welcome' })),
			mkLink('Admin', 'cog', serverInfoUrl()),
		];

		const appName = 'Varasto';

		return (
			<DefaultLayout
				appName={appName}
				appHomepage="https://function61.com/varasto"
				navLinks={navLinks}
				logoNode={
					<img
						src={globalConfig().assetsDir + '/../logo.svg'}
						title={appName}
						style={{ height: '40px' }}
					/>
				}
				logoClickUrl={browseUrl({ dir: RootFolderId })}
				breadcrumbs={this.props.breadcrumbs.concat({
					title: this.props.titleElem || this.props.title,
				})}
				content={this.props.children}
				version={version}
				pageTitle={this.props.title}
			/>
		);
	}
}
