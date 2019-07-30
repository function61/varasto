import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { Loading } from 'f61ui/component/loading';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { ClientCreate, ClientRemove } from 'generated/stoserver/stoservertypes_commands';
import { getClients } from 'generated/stoserver/stoservertypes_endpoints';
import { Client } from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface ClientsPageState {
	clients?: Client[];
}

export default class ClientsPage extends React.Component<{}, ClientsPageState> {
	state: ClientsPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<AppDefaultLayout title="Clients" breadcrumbs={[]}>
				{this.renderData()}
			</AppDefaultLayout>
		);
	}

	private renderData() {
		const clients = this.state.clients;

		if (!clients) {
			return <Loading />;
		}

		const toRow = (obj: Client) => (
			<tr key={obj.Id}>
				<td>{obj.Id}</td>
				<td>{obj.Name}</td>
				<td>
					<SecretReveal secret={obj.AuthToken} />
				</td>
				<td>
					<CommandIcon command={ClientRemove(obj.Id)} />
				</td>
			</tr>
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Id</th>
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
