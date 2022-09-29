/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package mrc

import (
	"fmt"

	"chainguard.dev/mrclean/pkg/attestation"
	"chainguard.dev/mrclean/pkg/sarif"
	"chainguard.dev/mrclean/pkg/vex"
)

type MRC struct {
	impl    Implementation
	Options Options
}

type Options struct {
	Products []string // List of products to match in CSAF docs
	Format   string   // Firmat of the vex documents
	Sign     bool     // When true, attestations will be signed before attaching
}

func New() *MRC {
	return &MRC{
		impl: &defaultMRCImplementation{},
	}
}

// ApplyFiles takes a list of paths to vex files and applies them to a report
func (mrc *MRC) ApplyFiles(r *sarif.Report, files []string) (*sarif.Report, error) {
	vexes, err := mrc.impl.OpenVexData(mrc.Options, files)
	if err != nil {
		return nil, fmt.Errorf("opening vex data: %w", err)
	}

	return mrc.Apply(r, vexes)
}

// Apply takes a sarif report and applies one or more vex documents
func (mrc *MRC) Apply(r *sarif.Report, vexDocs []*vex.VEX) (finalReport *sarif.Report, err error) {
	// Sort the docs by date
	vexDocs = mrc.impl.Sort(vexDocs)

	// Apply the sorted documents to the report
	for i, doc := range vexDocs {
		finalReport, err = mrc.impl.ApplySingleVEX(r, doc)
		if err != nil {
			return nil, fmt.Errorf("applying vex document #%d: %w", i, err)
		}
	}

	return finalReport, nil
}

// Generate an attestation from a VEX
func (mrc *MRC) Attest(vexDataPath string, imageRefs []string) (*attestation.Attestation, error) {
	doc, err := mrc.impl.OpenVexData(mrc.Options, []string{vexDataPath})
	if err != nil {
		return nil, fmt.Errorf("opening vex data")
	}

	// Generate the attestation
	att := attestation.New()
	att.Predicate = doc
	att.AddImageSubjects(imageRefs)

	return att, nil
}

// Attach attaches an attestation to a list of images
func (mrc *MRC) Attach(att *attestation.Attestation, imageRefs []string) (err error) {
	var attestationBytes []byte
	if mrc.Options.Sign {
		attestationBytes, err = mrc.impl.SignAttestation(att)
	} else {
		attestationBytes, err = mrc.impl.AttestationBytes(att)
	}
	if err != nil {
		return fmt.Errorf("signing attestation: %w", err)
	}

	fmt.Println(string(attestationBytes))
	return nil
}