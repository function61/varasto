import { Panel } from 'f61ui/component/bootstrap';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

export default class EncryptionKeysPage extends React.Component<{}, {}> {
	render() {
		return (
			<SettingsLayout title="Encryption keys" breadcrumbs={[]}>
				<Panel heading="Encryption keys">
					<p>todo</p>
				</Panel>
			</SettingsLayout>
		);
	}
}
