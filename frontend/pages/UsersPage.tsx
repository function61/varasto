import { Panel } from 'f61ui/component/bootstrap';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { Loading } from 'f61ui/component/loading';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { ClientCreate, ClientRemove } from 'generated/stoserver/stoservertypes_commands';
import { getClients } from 'generated/stoserver/stoservertypes_endpoints';
import { Client } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface UsersPageState {
	clients?: Client[];
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
		const clients = this.state.clients;

		if (!clients) {
			return <Loading />;
		}

		const toRow = (apiKey: Client) => (
			<tr key={apiKey.Id}>
				<td>{apiKey.Name}</td>
				<td>
					<Timestamp ts={apiKey.Created} />
				</td>
				<td>
					<SecretReveal secret={apiKey.AuthToken} />
				</td>
				<td>
					<CommandIcon command={ClientRemove(apiKey.Id)} />
				</td>
			</tr>
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Age</th>
						<th>Name</th>
						<th>AuthToken</th>
						<th />
					</tr>
				</thead>
				<tbody>{clients.map(toRow)}</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							<CommandButton command={ClientCreate()} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private async fetchData() {
		const clients = await getClients();

		this.setState({ clients });
	}
}
