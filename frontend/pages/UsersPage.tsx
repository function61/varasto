import { Panel } from 'f61ui/component/bootstrap';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { Loading } from 'f61ui/component/loading';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { ApikeyCreate, ApikeyRemove } from 'generated/stoserver/stoservertypes_commands'; // FIXME
import { getApiKeys } from 'generated/stoserver/stoservertypes_endpoints';
import { ApiKey } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface UsersPageState {
	apiKeys?: ApiKey[];
}

export default class UsersPage extends React.Component<{}, UsersPageState> {
	state: UsersPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<SettingsLayout title="Users" breadcrumbs={[]}>
				<Panel heading="Users">
					<p>todo</p>
				</Panel>

				<Panel heading="Encryption keys">
					<p>todo</p>
				</Panel>

				<Panel heading="API keys">{this.renderApiKeys()}</Panel>
			</SettingsLayout>
		);
	}

	private renderApiKeys() {
		const apiKeys = this.state.apiKeys;

		if (!apiKeys) {
			return <Loading />;
		}

		const toRow = (apiKey: ApiKey) => (
			<tr key={apiKey.Id}>
				<td>{apiKey.Name}</td>
				<td>
					<Timestamp ts={apiKey.Created} />
				</td>
				<td>
					<SecretReveal secret={apiKey.AuthToken} />
				</td>
				<td>
					<CommandIcon
						command={ApikeyRemove(apiKey.Id, { disambiguation: apiKey.Name })}
					/>
				</td>
			</tr>
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Name</th>
						<th>Age</th>
						<th>AuthToken</th>
						<th />
					</tr>
				</thead>
				<tbody>{apiKeys.map(toRow)}</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							<CommandButton command={ApikeyCreate()} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private async fetchData() {
		const apiKeys = await getApiKeys();

		this.setState({ apiKeys });
	}
}
