import { getCurrentLocation } from 'f61ui/browserutils';
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
		const currentLoc = getCurrentLocation();

		return (
			<div>
				<ul className="nav nav-tabs" role="tablist" style={{ marginBottom: '16px' }}>
					{this.props.tabs.map((tab) => (
						<li role="presentation" className={tab.url === currentLoc ? 'active' : ''}>
							<a href={tab.url}>{tab.title}</a>
						</li>
					))}
				</ul>
				<div className="tab-content">{this.props.children}</div>
			</div>
		);
	}
}
