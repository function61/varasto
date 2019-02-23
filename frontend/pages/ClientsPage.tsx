import { Loading } from 'f61ui/component/loading';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { getClients } from 'generated/bupserver_endpoints';
import { Client } from 'generated/bupserver_types';
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
			</tr>
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Id</th>
						<th>Name</th>
						<th>AuthToken</th>
					</tr>
				</thead>
				<tbody>{clients.map(toRow)}</tbody>
			</table>
		);
	}

	private async fetchData() {
		const clients = await getClients();

		this.setState({ clients });
	}
}
