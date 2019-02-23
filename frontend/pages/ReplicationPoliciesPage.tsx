import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { getReplicationPolicies } from 'generated/bupserver_endpoints';
import { ReplicationPolicy } from 'generated/bupserver_types';
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
			</tr>
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Id</th>
						<th>Name</th>
						<th>Desired volumes</th>
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
