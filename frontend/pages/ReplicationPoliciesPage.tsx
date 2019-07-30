import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { ReplicationpolicyChangeDesiredVolumes } from 'generated/stoserver/stoservertypes_commands';
import { getReplicationPolicies } from 'generated/stoserver/stoservertypes_endpoints';
import { ReplicationPolicy } from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface ReplicationPoliciesPageState {
	replicationpolicies?: ReplicationPolicy[];
}

export default class ReplicationPoliciesPage extends React.Component<
	{},
	ReplicationPoliciesPageState
> {
	state: ReplicationPoliciesPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<AppDefaultLayout title="Replication policies" breadcrumbs={[]}>
				{this.renderData()}
			</AppDefaultLayout>
		);
	}

	private renderData() {
		const replicationpolicies = this.state.replicationpolicies;

		if (!replicationpolicies) {
			return <Loading />;
		}

		const toRow = (obj: ReplicationPolicy) => (
			<tr key={obj.Id}>
				<td>{obj.Id}</td>
				<td>{obj.Name}</td>
				<td>{obj.DesiredVolumes.join(', ')}</td>
				<td>
					<Dropdown>
						<CommandLink command={ReplicationpolicyChangeDesiredVolumes(obj.Id)} />
					</Dropdown>
				</td>
			</tr>
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Id</th>
						<th>Name</th>
						<th>Desired volumes</th>
						<th />
					</tr>
				</thead>
				<tbody>{replicationpolicies.map(toRow)}</tbody>
			</table>
		);
	}

	private async fetchData() {
		const replicationpolicies = await getReplicationPolicies();

		this.setState({ replicationpolicies });
	}
}
