import { Panel } from 'f61ui/component/bootstrap';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

export default class UsersPage extends React.Component<{}, {}> {
	render() {
		return (
			<SettingsLayout title="Users" breadcrumbs={[]}>
				<Panel heading="Users">
					<p>todo</p>
				</Panel>
			</SettingsLayout>
		);
	}
}
