import { getCurrentHash } from 'f61ui/browserutils';
import * as React from 'react';

export interface Tab {
	url: string;
	title: string;
}

interface TabControllerProps {
	tabs: Tab[];
	children: React.ReactNode;
}

export class TabController extends React.Component<TabControllerProps, {}> {
	render() {
		const currentTabUrl = getCurrentHash();
		const activeTabs = this.props.tabs.filter((tab) => tab.url === currentTabUrl);

		return (
			<div>
				<ul className="nav nav-tabs" role="tablist" style={{ marginBottom: '16px' }}>
					{this.props.tabs.map((tab) => (
						<li
							role="presentation"
							className={
								activeTabs[0] && activeTabs[0].url === tab.url ? 'active' : ''
							}>
							<a href={tab.url}>{tab.title}</a>
						</li>
					))}
				</ul>
				<div className="tab-content">{this.props.children}</div>
			</div>
		);
	}
}
