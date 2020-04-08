import { Result } from 'f61ui/component/result';
import {
	changeSensitivity,
	getMaxSensitivityFromLocalStorage,
	Sensitivity,
	sensitivityLabel,
} from 'component/sensitivity';
import {
	DangerLabel,
	Glyphicon,
	Panel,
	SuccessLabel,
	WarningLabel,
	tableClassStripedHover,
} from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Timestamp } from 'f61ui/component/timestamp';
import { unrecognizedValue } from 'f61ui/utils';
import { SubsystemStart, SubsystemStop } from 'generated/stoserver/stoservertypes_commands';
import {
	getHealth,
	getServerInfo,
	getSubsystemStatuses,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	Health,
	HealthStatus,
	ServerInfo,
	SubsystemStatus,
} from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface ServerInfoPageState {
	serverInfo: Result<ServerInfo>;
	health: Result<Health>;
	subsystemStatuses: Result<SubsystemStatus[]>;
	currSens: Sensitivity;
}

export default class ServerInfoPage extends React.Component<{}, ServerInfoPageState> {
	state: ServerInfoPageState = {
		serverInfo: new Result<ServerInfo>((_) => {
			this.setState({ serverInfo: _ });
		}),
		health: new Result<Health>((_) => {
			this.setState({ health: _ });
		}),
		subsystemStatuses: new Result<SubsystemStatus[]>((_) => {
			this.setState({ subsystemStatuses: _ });
		}),
		currSens: getMaxSensitivityFromLocalStorage(),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		return (
			<SettingsLayout title="Server info &amp; health" breadcrumbs={[]}>
				<Panel heading="Server info">{this.renderInfo()}</Panel>
				<Panel heading="Subsystems">{this.renderSubsystems()}</Panel>
				<Panel heading="Health">{this.renderHealth()}</Panel>
				<Panel heading="Sensitivity">{this.renderSensitivitySelector()}</Panel>
			</SettingsLayout>
		);
	}

	private renderInfo() {
		const [serverInfo, loadingOrError] = this.state.serverInfo.unwrap();

		interface Item {
			h: string;
			t: React.ReactNode;
		}

		const items: Item[] = serverInfo
			? [
					{ h: 'Varasto version', t: serverInfo.AppVersion },
					{ h: 'Varasto uptime', t: <Timestamp ts={serverInfo.StartedAt} /> },
					{ h: 'Database size', t: bytesToHumanReadable(serverInfo.DatabaseSize) },
					{ h: 'Go version', t: serverInfo.GoVersion },
					{
						h: 'Server OS / arch',
						t: `${serverInfo.ServerOs} / ${serverInfo.ServerArch}`,
					},
					{ h: 'Process ID', t: serverInfo.ProcessId },
					{ h: 'CPU count', t: serverInfo.CpuCount.toString() },
					{ h: 'Goroutines', t: serverInfo.Goroutines.toString() },
					{ h: 'Heap memory', t: bytesToHumanReadable(serverInfo.HeapBytes) },
			  ]
			: [];

		return (
			<table className={tableClassStripedHover}>
				<tbody>
					{items.map((item) => (
						<tr key={item.h}>
							<th>{item.h}</th>
							<td>{item.t}</td>
						</tr>
					))}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={2}>{loadingOrError}</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private renderHealth() {
		const [health, loadingOrError] = this.state.health.unwrap();

		const rows: JSX.Element[] = [];

		const pushHealthNodeAsRow = (node: Health, indentLevel: number) => {
			rows.push(
				<tr>
					<td>{healthStatusToIcon(node.Health)}</td>
					<td style={{ paddingLeft: indentLevel * 32 + 'px' }}>{node.Title}</td>
					<td>{node.Details}</td>
				</tr>,
			);

			node.Children.forEach((childHealth) => {
				pushHealthNodeAsRow(childHealth, indentLevel + 1);
			});
		};

		if (health) {
			pushHealthNodeAsRow(health, 0);
		}

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th />
						<th>Title</th>
						<th>Details</th>
					</tr>
				</thead>
				<tbody>{rows}</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>{loadingOrError}</td>
					</tr>
				</tfoot>
			</table>
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

	private renderSensitivitySelector() {
		const sensitivityRadioChange = (e: React.ChangeEvent<HTMLInputElement>) => {
			changeSensitivity(+e.target.value);
			this.setState({ currSens: getMaxSensitivityFromLocalStorage() });
		};

		const oneSensitivityRadio = (sens: Sensitivity) => (
			<div key={sens}>
				<label>
					<input
						type="radio"
						name="changeSensitivityRadio"
						onChange={sensitivityRadioChange}
						value={sens}
						checked={sens === this.state.currSens}
					/>{' '}
					{sensitivityLabel(sens)}
				</label>
			</div>
		);

		return (
			<div>
				{oneSensitivityRadio(Sensitivity.FamilyFriendly)}
				{oneSensitivityRadio(Sensitivity.Sensitive)}
				{oneSensitivityRadio(Sensitivity.MyEyesOnly)}
			</div>
		);
	}

	private fetchData() {
		this.state.serverInfo.load(() => getServerInfo());
		this.state.health.load(() => getHealth());
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

function healthStatusToIcon(health: HealthStatus): JSX.Element {
	switch (health) {
		case HealthStatus.Fail:
			return (
				<DangerLabel>
					<Glyphicon icon="fire" />
				</DangerLabel>
			);
		case HealthStatus.Warn:
			return (
				<WarningLabel>
					<Glyphicon icon="warning-sign" />
				</WarningLabel>
			);
		case HealthStatus.Pass:
			return (
				<SuccessLabel>
					<Glyphicon icon="ok" />
				</SuccessLabel>
			);
		default:
			throw unrecognizedValue(health);
	}
}
