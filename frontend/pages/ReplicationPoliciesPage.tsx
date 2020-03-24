import { thousandSeparate } from 'component/numberformatter';
import { Result } from 'component/result';
import { DangerLabel, DefaultLabel, Panel } from 'f61ui/component/bootstrap';
import { CommandButton, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import {
	DatabaseDiscoverReconcilableReplicationPolicies,
	DatabaseReconcileOutOfSyncDesiredVolumes,
	DatabaseReconcileReplicationPolicy,
	ReplicationpolicyChangeDesiredVolumes,
} from 'generated/stoserver/stoservertypes_commands';
import {
	getReconcilableItems,
	getReplicationPolicies,
	getVolumes,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	ReconciliationReport,
	ReplicationPolicy,
	Volume,
} from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface ReplicationPoliciesPageState {
	selectedCollIds: string[];
	replicationpolicies: Result<ReplicationPolicy[]>;
	reconciliationReport: Result<ReconciliationReport>;
	volumes: Result<Volume[]>;
}

export default class ReplicationPoliciesPage extends React.Component<
	{},
	ReplicationPoliciesPageState
> {
	state: ReplicationPoliciesPageState = {
		selectedCollIds: [],
		reconciliationReport: new Result<ReconciliationReport>((_) => {
			this.setState({ reconciliationReport: _ });
		}),
		replicationpolicies: new Result<ReplicationPolicy[]>((_) => {
			this.setState({ replicationpolicies: _ });
		}),
		volumes: new Result<Volume[]>((_) => {
			this.setState({ volumes: _ });
		}),
	};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<SettingsLayout title="Replication policies" breadcrumbs={[]}>
				<Panel heading="Policies">{this.renderPolicies()}</Panel>

				<Panel heading="Reconciliation">{this.renderReconcilable()}</Panel>
			</SettingsLayout>
		);
	}

	private renderPolicies() {
		const [replicationpolicies, loadingOrError] = this.state.replicationpolicies.unwrap();

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
				<tbody>
					{(replicationpolicies || []).map((rp) => (
						<tr key={rp.Id}>
							<td>{rp.Id}</td>
							<td>{rp.Name}</td>
							<td>{rp.DesiredVolumes.join(', ')}</td>
							<td>
								<Dropdown>
									<CommandLink
										command={ReplicationpolicyChangeDesiredVolumes(rp.Id)}
									/>
								</Dropdown>
							</td>
						</tr>
					))}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>{loadingOrError}</td>
					</tr>
				</tfoot>
			</table>
		);
	}
	private renderReconcilable() {
		const [report, volumes, loadingOrError] = Result.unwrap2(
			this.state.reconciliationReport,
			this.state.volumes,
		);

		if (!report || !volumes) {
			return loadingOrError;
		}

		const masterCheckedChange = () => {
			const selectedCollIds = report.Items.map((item) => item.CollectionId);

			this.setState({ selectedCollIds });
		};

		const collCheckedChange = (e: React.ChangeEvent<HTMLInputElement>) => {
			const collId = e.target.value;

			// removes collId if it already exists
			const selectedCollIds = this.state.selectedCollIds.filter((id) => id !== collId);

			if (e.target.checked) {
				selectedCollIds.push(collId);
			}

			this.setState({ selectedCollIds });
		};

		return (
			<div>
				<p>
					{thousandSeparate(report.TotalItems)} collections in non-compliance with its
					replication policy.
				</p>

				<table className="table table-striped table-hover">
					<thead>
						<tr>
							<th>
								<input type="checkbox" onChange={masterCheckedChange} />
							</th>
							<th>Collection</th>
							<th>Blobs</th>
							<th>Problems</th>
							<th colSpan={2}>
								Replicas
								<br />
								Desired &nbsp; &nbsp;Reality
							</th>
						</tr>
					</thead>
					<tbody>
						{report.Items.map((r) => (
							<tr>
								<td>
									<input
										type="checkbox"
										checked={
											this.state.selectedCollIds.indexOf(r.CollectionId) !==
											-1
										}
										onChange={collCheckedChange}
										value={r.CollectionId}
									/>
								</td>
								<td>{r.Description}</td>
								<td>{thousandSeparate(r.TotalBlobs)}</td>
								<td>
									{r.ProblemRedundancy && <DangerLabel>redundancy</DangerLabel>}
									{r.ProblemDesiredReplicasOutdated && (
										<DangerLabel>desiredVolsOutOfSync</DangerLabel>
									)}
								</td>
								<td>{r.DesiredReplicaCount}</td>
								<td>
									{r.ReplicaStatuses.sort(
										(a, b) => b.BlobCount - a.BlobCount,
									).map((rs) => {
										const vol = volumes.filter((v) => v.Id === rs.Volume);
										const volLabel =
											vol.length === 1 ? vol[0].Label : '(error)';

										if (rs.BlobCount === r.TotalBlobs) {
											return (
												<span className="margin-left">
													<DefaultLabel title={rs.BlobCount.toString()}>
														{volLabel}
													</DefaultLabel>
												</span>
											);
										} else {
											return (
												<span className="margin-left">
													<DangerLabel title={rs.BlobCount.toString()}>
														{volLabel}
													</DangerLabel>
												</span>
											);
										}
									})}
								</td>
							</tr>
						))}
					</tbody>
					<tfoot>
						<tr>
							<td colSpan={2}>
								{this.state.selectedCollIds.length > 0 && (
									<div>
										<CommandButton
											command={DatabaseReconcileReplicationPolicy(
												this.state.selectedCollIds.join(','),
											)}
										/>
										<CommandButton
											command={DatabaseReconcileOutOfSyncDesiredVolumes(
												this.state.selectedCollIds.join(','),
											)}
										/>
									</div>
								)}
							</td>
							<td colSpan={99}>
								{thousandSeparate(
									report.Items.reduce(
										(prev, current) => prev + current.TotalBlobs,
										0,
									),
								)}
							</td>
						</tr>
					</tfoot>
				</table>

				<CommandButton command={DatabaseDiscoverReconcilableReplicationPolicies()} />
			</div>
		);
	}

	private async fetchData() {
		this.state.replicationpolicies.load(() => getReplicationPolicies());
		this.state.reconciliationReport.load(() => getReconcilableItems());
		this.state.volumes.load(() => getVolumes());
	}
}
