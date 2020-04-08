import { Result } from 'f61ui/component/result';
import {
	DangerLabel,
	Panel,
	SuccessLabel,
	WarningLabel,
	tableClassStripedHover,
} from 'f61ui/component/bootstrap';
import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Timestamp } from 'f61ui/component/timestamp';
import { SubsystemStart, SubsystemStop } from 'generated/stoserver/stoservertypes_commands';
import { getSubsystemStatuses } from 'generated/stoserver/stoservertypes_endpoints';
import { SubsystemStatus } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
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
			<SettingsLayout title="Subsystems" breadcrumbs={[]}>
				<Panel heading="Subsystems">{this.renderSubsystems()}</Panel>
			</SettingsLayout>
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
