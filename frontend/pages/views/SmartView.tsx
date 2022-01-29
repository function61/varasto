import { DocLink } from 'component/doclink';
import { thousandSeparate } from 'component/numberformatter';
import { InfoAlert } from 'f61ui/component/alerts';
import {
	DangerLabel,
	SuccessLabel,
	DefaultLabel,
	CollapsePanel,
	Glyphicon,
	tableClassStripedHover,
} from 'f61ui/component/bootstrap';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { Info } from 'f61ui/component/info';
import { Timestamp } from 'f61ui/component/timestamp';
import { NodeSmartScan, VolumeSmartSetId } from 'generated/stoserver/stoservertypes_commands';
import { DocRef, Volume } from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

interface SmartViewProps {
	volumes: Volume[];
}

interface SmartViewState {
	rawReport?: string;
}

export default class SmartView extends React.Component<SmartViewProps, SmartViewState> {
	state: SmartViewState = {};

	render() {
		return (
			<div>
				{this.reports()}

				{this.state.rawReport ? (
					<span style={{ whiteSpace: 'pre' }}>{this.state.rawReport}</span>
				) : null}

				{this.configurator()}
			</div>
		);
	}

	private reports() {
		const volumesWithSmart = this.props.volumes.filter(
			(vol) => vol.Smart.LatestReport !== null && vol.Smart.Id !== '',
		);

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th>Passed</th>
						<th>Label</th>
						<th>Description</th>
						<th>Reported</th>
						<th>Temperature</th>
						<th>PowerCycleCount</th>
						<th>PowerOnTime</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{volumesWithSmart.map((vol) => {
						const smart = vol.Smart.LatestReport!;

						return (
							<tr key={vol.Id}>
								<td>
									{smart.Passed ? (
										<SuccessLabel title="Pass">✓</SuccessLabel>
									) : (
										<DangerLabel title="Fail">❌</DangerLabel>
									)}
								</td>
								<td>{vol.Label}</td>
								<td>{vol.Description}</td>
								<td>
									<Timestamp ts={smart.Time} />
								</td>
								<td>
									{smart.Temperature
										? smart.Temperature.toString() + ' °C'
										: null}
								</td>
								<td>
									{smart.PowerCycleCount
										? thousandSeparate(smart.PowerCycleCount)
										: null}
								</td>
								<td>
									{smart.PowerOnTime ? thousandSeparate(smart.PowerOnTime) : null}
								</td>
								<td>
									<Glyphicon
										icon="search"
										click={() => {
											this.setState({
												rawReport: smart.RawReport || undefined,
											});
										}}
									/>
								</td>
							</tr>
						);
					})}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							{volumesWithSmart.length === 0 && (
								<div>
									<InfoAlert>
										No SMART-reporting volumes found. Read docs first:{' '}
										<DocLink doc={DocRef.DocsUsingSmartMonitoringIndexMd} />
									</InfoAlert>
								</div>
							)}
							<CommandButton command={NodeSmartScan()} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private configurator() {
		return (
			<CollapsePanel
				heading={
					this.props.volumes.filter((vol) => !vol.Smart.Id).length +
					' volume(s) without SMART configured'
				}
				visualStyle="info">
				<table className={tableClassStripedHover}>
					<thead>
						<tr>
							<th>
								SMART polling enabled <Info text="Enable by specifying SMART ID" />
							</th>
							<th>Volume</th>
							<th>SMART ID</th>
						</tr>
					</thead>
					<tbody>
						{this.props.volumes.map((vol) => (
							<tr key={vol.Uuid}>
								<td>
									{vol.Smart.Id ? (
										<SuccessLabel>Yes</SuccessLabel>
									) : (
										<DefaultLabel>No</DefaultLabel>
									)}
								</td>
								<td>{vol.Label}</td>
								<td>
									{vol.Smart.Id}{' '}
									<CommandIcon
										command={VolumeSmartSetId(
											vol.Id,
											vol.Smart.Id,
											vol.Smart.Backend!,
											{
												disambiguation: vol.Label,
											},
										)}
									/>
								</td>
							</tr>
						))}
					</tbody>
				</table>
			</CollapsePanel>
		);
	}
}
