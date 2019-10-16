import { getCurrentHash } from 'f61ui/browserutils';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { NavLink } from 'f61ui/component/navigation';
import { DefaultLayout } from 'f61ui/layout/defaultlayout';
import { RootFolderId } from 'generated/stoserver/stoservertypes_types';
import { version } from 'generated/version';
import * as React from 'react';
import { browseRoute, serverInfoRoute } from 'routes';

interface AppDefaultLayoutProps {
	title: string;
	titleElem?: React.ReactNode;
	breadcrumbs: Breadcrumb[];
	children: React.ReactNode;
}

// app's default layout uses the default layout with props that are common to the whole app
export class AppDefaultLayout extends React.Component<AppDefaultLayoutProps, {}> {
	render() {
		const hash = getCurrentHash();

		const navLinks: NavLink[] = [
			{
				title: 'Browse',
				url: browseRoute.buildUrl({ dir: RootFolderId, view: '' }),
				active: browseRoute.matchUrl(hash) !== null,
			},
			{
				title: 'Settings',
				url: serverInfoRoute.buildUrl({}),
				active: serverInfoRoute.matchUrl(hash) !== null,
			},
		];

		return (
			<DefaultLayout
				appName="Varasto"
				appHomepage="https://github.com/function61/varasto"
				navLinks={navLinks}
				logoUrl={browseRoute.buildUrl({ dir: RootFolderId, view: '' })}
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
