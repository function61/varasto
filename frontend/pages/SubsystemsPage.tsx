import { RefreshButton } from 'component/refreshbutton';
import { Result } from 'f61ui/component/result';
import { DocLink } from 'component/doclink';
import {
	DangerLabel,
	Panel,
	CollapsePanel,
	SuccessLabel,
	WarningLabel,
	tableClassStripedHover,
} from 'f61ui/component/bootstrap';
import { CommandLink } from 'f61ui/component/CommandButton';
import { fuseServerUrl } from 'generated/stoserver/stoserverui_uiroutes';
import { Dropdown } from 'f61ui/component/dropdown';
import { Timestamp } from 'f61ui/component/timestamp';
import { SubsystemStart, SubsystemStop } from 'generated/stoserver/stoservertypes_commands';
import { getSubsystemStatuses } from 'generated/stoserver/stoservertypes_endpoints';
import { SubsystemStatus, DocRef } from 'generated/stoserver/stoservertypes_types';
import { AdminLayout } from 'layout/AdminLayout';
import * as React from 'react';

interface ServerInfoPageState {
	subsystemStatuses: Result<SubsystemStatus[]>;
}

export default class ServerInfoPage extends React.Component<{}, ServerInfoPageState> {
	state: ServerInfoPageState = {
		subsystemStatuses: new Result<SubsystemStatus[]>((_) => {
			this.setState({ subsystemStatuses: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		return (
			<AdminLayout title="Subsystems" breadcrumbs={[]}>
				<Panel
					heading={
						<div>
							Subsystems{' '}
							<RefreshButton
								refresh={() => {
									this.fetchData();
								}}
							/>
						</div>
					}>
					{this.renderSubsystems()}
				</Panel>
				<CollapsePanel heading="Info">{this.info()}</CollapsePanel>
			</AdminLayout>
		);
	}

	private renderSubsystems() {
		const [subsystemStatuses, loadingOrError] = this.state.subsystemStatuses.unwrap();

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th>Status</th>
						<th>Subsystem</th>
						<th>Started</th>
						<th>Process ID</th>
						<th>HTTP mount</th>
						<th />
					</tr>
				</thead>
				<tbody>
					{(subsystemStatuses || []).map((subsys) => {
						const started = subsys.Started; // TS doesn't remove null without this

						return (
							<tr key={subsys.Id}>
								<td>{subsystemStatusLabel(subsys.Alive, subsys.Enabled)}</td>
								<td>{subsys.Description}</td>
								<td>{started && <Timestamp ts={started} />}</td>
								<td>{subsys.Pid}</td>
								<td>{subsys.HttpMount}</td>
								<td>
									<Dropdown>
										{!subsys.Enabled && (
											<CommandLink command={SubsystemStart(subsys.Id)} />
										)}
										{subsys.Enabled && (
											<CommandLink command={SubsystemStop(subsys.Id)} />
										)}
									</Dropdown>
								</td>
							</tr>
						);
					})}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>{loadingOrError}</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private fetchData() {
		this.state.subsystemStatuses.load(() => getSubsystemStatuses());
	}

	private info() {
		return (
			<div>
				<p>
					Subsystems are semi-autonomous components of Varasto, that by default run as
					child processes of Varasto's main server - but in advanced use cases can be ran
					independently over HTTP on a different machine (or container) if desired.
				</p>
				<p>
					We expect that most users won't care about customizing this and just use the
					default child-process configuration.
				</p>
				<p>Example use cases include:</p>
				<ul>
					<li>
						Shed load from one server and move thumbnailing/transcoding subsystems into
						different servers - you can even use multiple instances behind
						loadbalancers!
					</li>
					<li>
						Normally you need to give Varasto's container privileged access so it can
						query SMART data for your disks. You could run the SMART subsystem in a
						different container and now you don't need to give Varasto main server's
						container those raw disk access privileges - which reduces overall attack
						surface.
					</li>
					<li>
						<a href={fuseServerUrl()}>Network folders</a> requires FUSE projector which
						is Linux-only. If you wish to run Varasto server on Windows, you can run
						FUSE projector on a separate Linux server.{' '}
						<DocLink doc={DocRef.DocsDataInterfacesNetworkFoldersIndexMd} />
					</li>
				</ul>
			</div>
		);
	}
}

function subsystemStatusLabel(alive: boolean, enabled: boolean): React.ReactNode {
	if (alive && enabled) {
		return <SuccessLabel>running</SuccessLabel>;
	} else if (alive && !enabled) {
		return <DangerLabel>running but should be stopped</DangerLabel>;
	} else if (!alive && enabled) {
		return <DangerLabel>stopped but should be running</DangerLabel>;
	} else if (!alive && !enabled) {
		return <WarningLabel>stopped</WarningLabel>;
	} else {
		throw new Error('Should not happen');
	}
}
